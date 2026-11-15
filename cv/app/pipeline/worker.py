"""Camera worker: capture -> detect -> occupancy -> backend events."""

from __future__ import annotations

import logging
import threading
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import cv2
import numpy as np

from ..capture.source import SourceError, VideoSource
from ..core.config import REPO_ROOT, Settings
from ..detection.detector import Detection, VehicleDetector
from ..occupancy.geometry import normalize_box, parse_polygon
from ..occupancy.tracker import OccupancyTracker, Spot, StatusChange
from ..schemas.events import SpotEventPayload
from ..storage.backend import BackendClient, BackendError

logger = logging.getLogger(__name__)

STATUS_COLORS = {
    "free": (60, 200, 60),
    "occupied": (40, 40, 220),
    "unknown": (150, 150, 150),
}


class CameraWorker:
    def __init__(self, settings: Settings, backend: BackendClient | None) -> None:
        self.settings = settings
        self.backend = backend

        self._stop = threading.Event()
        self._thread: threading.Thread | None = None
        self._lock = threading.Lock()

        self._status = "idle"
        self._error: str | None = None
        self._source: VideoSource | None = None
        self._tracker: OccupancyTracker | None = None
        self._overlay: cv2.VideoWriter | None = None
        self._overlay_path: Path | None = None
        self._pending: list[SpotEventPayload] = []

        self._frame_index = 0
        self._frames_processed = 0
        self._last_frame_at: datetime | None = None
        self._last_heartbeat = 0.0
        self._measured_fps = 0.0
        self._measure_started = time.monotonic()
        self._measured_frames = 0

    # ---------------------------------------------------------------- lifecycle
    def start(self) -> None:
        if self._thread and self._thread.is_alive():
            return
        self._thread = threading.Thread(
            target=self.run,
            name=f"camera-{self.settings.camera_id}",
            daemon=True,
        )
        self._thread.start()

    def stop(self) -> None:
        self._stop.set()
        if self._thread and self._thread.is_alive():
            self._thread.join(timeout=10)
        self._close_overlay()

    def is_alive(self) -> bool:
        return bool(self._thread and self._thread.is_alive())

    def run(self) -> None:
        self._set_status("starting")
        try:
            self._tracker = self._build_tracker()
            self._source = VideoSource.open(
                self.settings.source_type,
                self.settings.source or "",
            )
            detector = VehicleDetector(
                self.settings.model_path,
                confidence=min(self.settings.confidence, self.settings.weak_confidence),
                device=self.settings.device,
            )
            self._heartbeat("online")
            self._set_status("running")
            self._loop(detector)
        except Exception as exc:  # noqa: BLE001 - worker must report any failure
            logger.exception("camera worker failed")
            with self._lock:
                self._error = str(exc)
            self._set_status("error")
        finally:
            self._release_source()
            self._heartbeat("offline")
            self._close_overlay()
            with self._lock:
                if self._status != "error":
                    self._status = "stopped"
            if self.backend is not None:
                self.backend.close()

    # ------------------------------------------------------------------- loop
    def _loop(self, detector: VehicleDetector) -> None:
        source = self._source
        tracker = self._tracker
        if source is None or tracker is None:
            raise RuntimeError("worker is not initialised")

        native_fps = source.fps if source.fps > 0 else 25.0
        step = max(1, round(native_fps / self.settings.sample_fps))
        video_started = time.monotonic()
        logger.info(
            "worker loop: source=%s native_fps=%.1f sample_fps=%.1f step=%d",
            source.kind,
            native_fps,
            self.settings.sample_fps,
            step,
        )

        while not self._stop.is_set():
            frame = source.read()
            if frame is None:
                if source.is_file:
                    if self.settings.loop_video:
                        logger.info("video finished, looping")
                        source.reopen()
                        video_started = time.monotonic()
                        self._frame_index = 0
                        continue
                    logger.info("video finished")
                    break

                logger.warning("source lost, reconnecting in %.1fs", self.settings.reconnect_seconds)
                self._heartbeat("offline")
                time.sleep(self.settings.reconnect_seconds)
                try:
                    source.reopen()
                    self._heartbeat("online")
                except SourceError as exc:
                    logger.warning("reconnect failed: %s", exc)
                continue

            if self._frame_index % step == 0:
                frame = self._resize(frame)
                self._process(frame, detector, tracker)

            self._frame_index += 1

            if self.settings.realtime and source.is_file:
                target = self._frame_index / native_fps
                elapsed = time.monotonic() - video_started
                if target > elapsed:
                    time.sleep(target - elapsed)

    def _process(
        self,
        frame: np.ndarray,
        detector: VehicleDetector,
        tracker: OccupancyTracker,
    ) -> None:
        now = datetime.now(timezone.utc)
        height, width = frame.shape[:2]

        detections = detector.detect(frame)
        pairs = [
            (normalize_box(detection.box, width, height), detection.confidence)
            for detection in detections
        ]
        changes = tracker.update(pairs, at=now)

        annotated = frame.copy()
        self._draw(annotated, detections, tracker)

        if changes:
            for change in changes:
                logger.info(
                    "spot %s -> %s (confidence=%s)",
                    change.code,
                    change.status,
                    change.confidence,
                )
                self._pending.append(
                    SpotEventPayload(
                        spot_id=change.spot_id,
                        status=change.status,  # type: ignore[arg-type]
                        confidence=change.confidence,
                        captured_at=change.captured_at,
                        snapshot_path=self._save_snapshot(annotated, change),
                    )
                )

        self._write_overlay(annotated)

        with self._lock:
            self._frames_processed += 1
            self._measured_frames += 1
            self._last_frame_at = now

        self._flush_events()
        self._maybe_heartbeat()

    # --------------------------------------------------------------- helpers
    def _build_tracker(self) -> OccupancyTracker:
        if self.backend is None or self.settings.lot_id is None:
            raise RuntimeError("backend and lot_id are required")

        spots_raw = self._load_spots()
        spots: list[Spot] = []
        for raw in spots_raw:
            camera_id = raw.get("camera_id")
            if camera_id is not None and self.settings.camera_id not in (None, camera_id):
                continue
            try:
                polygon = parse_polygon(raw["polygon"])
            except (KeyError, TypeError, ValueError, IndexError) as exc:
                logger.warning("skip spot %s: bad polygon (%s)", raw.get("code"), exc)
                continue
            if len(polygon) < 3:
                logger.warning("skip spot %s: polygon has <3 points", raw.get("code"))
                continue
            spots.append(
                Spot(
                    id=int(raw["id"]),
                    code=str(raw.get("code", raw["id"])),
                    polygon=polygon,
                    initial_status=str(raw.get("status") or "unknown"),
                )
            )

        if not spots:
            raise RuntimeError(f"no spots configured for lot {self.settings.lot_id}")

        logger.info(
            "loaded %d spots: %s",
            len(spots),
            ", ".join(spot.code for spot in spots),
        )
        return OccupancyTracker(
            spots=spots,
            occupy_frames=self.settings.occupy_frames,
            free_frames=self.settings.free_frames,
            occupied_confidence=self.settings.confidence,
            weak_confidence=self.settings.weak_confidence,
        )

    def _load_spots(self) -> list[dict[str, Any]]:
        assert self.backend is not None
        assert self.settings.lot_id is not None
        while not self._stop.is_set():
            try:
                return self.backend.get_spots(self.settings.lot_id)
            except Exception as exc:  # noqa: BLE001 - retry until the backend is up
                logger.warning("cannot load spots yet: %s", exc)
                time.sleep(min(self.settings.reconnect_seconds, 5.0))
        raise RuntimeError("stopped before spots were loaded")

    def _resize(self, frame: np.ndarray) -> np.ndarray:
        width = frame.shape[1]
        if width <= self.settings.max_frame_width:
            return frame
        scale = self.settings.max_frame_width / width
        height = int(frame.shape[0] * scale)
        return cv2.resize(frame, (self.settings.max_frame_width, height))

    def _draw(
        self,
        frame: np.ndarray,
        detections: list[Detection],
        tracker: OccupancyTracker,
    ) -> None:
        height, width = frame.shape[:2]
        for state in tracker.states.values():
            points = np.array(
                [[int(x * width), int(y * height)] for x, y in state.spot.polygon],
                dtype=np.int32,
            )
            color = STATUS_COLORS.get(state.status, STATUS_COLORS["unknown"])
            cv2.polylines(frame, [points], True, color, 2)
            label = f"{state.spot.code}: {state.status}"
            x, y = points[0]
            cv2.putText(
                frame,
                label,
                (int(x), max(18, int(y) - 6)),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.55,
                color,
                2,
                cv2.LINE_AA,
            )

        for detection in detections:
            x1, y1, x2, y2 = (int(value) for value in detection.box)
            cv2.rectangle(frame, (x1, y1), (x2, y2), (0, 220, 255), 1)

    def _save_snapshot(self, frame: np.ndarray, change: StatusChange) -> str | None:
        if not self.settings.snapshot_dir:
            return None
        self.settings.snapshot_dir.mkdir(parents=True, exist_ok=True)
        stamp = change.captured_at.strftime("%Y%m%dT%H%M%S%f")
        path = self.settings.snapshot_dir / f"camera{self.settings.camera_id}_spot{change.spot_id}_{stamp}.jpg"
        if not cv2.imwrite(str(path), frame):
            logger.warning("cannot write snapshot %s", path)
            return None
        try:
            return str(path.relative_to(REPO_ROOT))
        except ValueError:
            return str(path)

    def _write_overlay(self, frame: np.ndarray) -> None:
        if not self.settings.save_overlay:
            return
        if self._overlay is None:
            self._init_overlay(frame)

        if self._overlay is not None:
            self._overlay.write(frame)

    def _init_overlay(self, frame: np.ndarray) -> None:
        height, width = frame.shape[:2]
        base = self.settings.overlay_path
        path = base.with_name(f"{base.stem}_camera{self.settings.camera_id}{base.suffix}")
        path.parent.mkdir(parents=True, exist_ok=True)
        writer = cv2.VideoWriter(
            str(path),
            cv2.VideoWriter_fourcc(*"mp4v"),
            max(1.0, self.settings.sample_fps),
            (width, height),
        )
        if not writer.isOpened():
            logger.warning("cannot open overlay video: %s", path)
            return
        self._overlay = writer
        self._overlay_path = path
        logger.info("writing annotated video to %s", path)

    def _close_overlay(self) -> None:
        if self._overlay is not None:
            self._overlay.release()
            self._overlay = None

    def _flush_events(self) -> None:
        if not self._pending or self.backend is None or self.settings.camera_id is None:
            return
        batch = self._pending[:500]
        try:
            self.backend.send_events(self.settings.camera_id, batch)
        except BackendError as exc:
            logger.warning("cannot send %d events: %s", len(batch), exc)
            self._pending = self._pending[-500:]
            return
        del self._pending[: len(batch)]

    def _maybe_heartbeat(self) -> None:
        now = time.monotonic()
        if now - self._last_heartbeat < self.settings.heartbeat_seconds:
            return

        elapsed = now - self._measure_started
        if elapsed >= 1.0:
            self._measured_fps = self._measured_frames / elapsed
            self._measured_frames = 0
            self._measure_started = now

        self._heartbeat("online")

    def _heartbeat(self, status: str) -> None:
        if self.backend is None or self.settings.camera_id is None:
            return
        try:
            self.backend.send_heartbeat(
                camera_id=self.settings.camera_id,
                status=status,
                last_frame_at=self._last_frame_at,
                fps=max(1, round(self._measured_fps or self.settings.sample_fps)),
                resolution=self._source.resolution() if self._source else None,
            )
        except BackendError as exc:
            logger.warning("heartbeat failed: %s", exc)
        self._last_heartbeat = time.monotonic()

    def _release_source(self) -> None:
        if self._source is not None:
            self._source.release()
            self._source = None

    def _set_status(self, status: str) -> None:
        with self._lock:
            self._status = status

    def snapshot(self) -> dict[str, Any]:
        with self._lock:
            spots = {}
            if self._tracker is not None:
                spots = {
                    state.spot.code: {
                        "spot_id": state.spot.id,
                        "status": state.status,
                        "confidence": state.confidence,
                    }
                    for state in self._tracker.states.values()
                }
            return {
                "status": self._status,
                "error": self._error,
                "camera_id": self.settings.camera_id,
                "lot_id": self.settings.lot_id,
                "source": self.settings.source,
                "source_type": self._source.kind if self._source else None,
                "frames_processed": self._frames_processed,
                "last_frame_at": self._last_frame_at.isoformat() if self._last_frame_at else None,
                "measured_fps": round(self._measured_fps, 2),
                "spots": spots,
                "pending_events": len(self._pending),
                "overlay_path": str(self._overlay_path) if self._overlay_path else None,
            }
