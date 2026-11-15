"""CV service configuration loaded from environment variables."""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from dotenv import load_dotenv

CV_ROOT = Path(__file__).resolve().parents[2]
REPO_ROOT = CV_ROOT.parent


def _load_env() -> None:
    for candidate in (REPO_ROOT / ".env", CV_ROOT / ".env", Path.cwd() / ".env"):
        if candidate.exists():
            load_dotenv(candidate, override=False)


def _env(name: str, default: str = "") -> str:
    return os.getenv(name, default).strip()


def _int(name: str, default: int) -> int:
    raw = _env(name)
    return int(raw) if raw else default


def _float(name: str, default: float) -> float:
    raw = _env(name)
    return float(raw) if raw else default


def _bool(name: str, default: bool) -> bool:
    raw = _env(name).lower()
    if not raw:
        return default
    return raw in {"1", "true", "yes", "on"}


def _optional_int(name: str) -> int | None:
    raw = _env(name)
    return int(raw) if raw else None


def _path(name: str, default: str, base: Path) -> Path:
    raw = _env(name) or default
    path = Path(raw).expanduser()
    return path if path.is_absolute() else (base / path).resolve()


@dataclass(frozen=True)
class Settings:
    app_env: str
    host: str
    port: int
    backend_url: str
    internal_token: str

    camera_id: int | None
    lot_id: int | None
    source: str | None
    source_type: str

    model_path: Path
    device: str
    confidence: float
    weak_confidence: float
    sample_fps: float
    max_frame_width: int
    occupy_frames: int
    free_frames: int

    realtime: bool
    loop_video: bool
    reconnect_seconds: float
    heartbeat_seconds: float

    save_overlay: bool
    overlay_path: Path
    snapshot_dir: Path
    log_level: str

    @property
    def can_start(self) -> bool:
        return bool(self.source and self.camera_id and self.lot_id and self.internal_token)

    @property
    def dry_run(self) -> bool:
        return not (self.backend_url and self.internal_token)


def load_settings() -> Settings:
    _load_env()

    return Settings(
        app_env=_env("APP_ENV", "dev"),
        host=_env("CV_HOST", "0.0.0.0"),
        port=_int("CV_PORT", 8000),
        backend_url=_env("CV_BACKEND_URL", "http://localhost:8080"),
        internal_token=_env("INTERNAL_TOKEN"),
        camera_id=_optional_int("CV_CAMERA_ID"),
        lot_id=_optional_int("CV_LOT_ID"),
        source=_env("CV_SOURCE") or None,
        source_type=_env("CV_SOURCE_TYPE", "auto"),
        model_path=_path("CV_MODEL_PATH", "models/yolo11n.pt", CV_ROOT),
        device=_env("CV_DEVICE", "auto"),
        confidence=_float("CONFIDENCE_THRESHOLD", 0.4),
        weak_confidence=_float("WEAK_CONFIDENCE_THRESHOLD", 0.2),
        sample_fps=max(0.5, _float("SAMPLE_FPS", 3.0)),
        max_frame_width=_int("MAX_FRAME_WIDTH", 1280),
        occupy_frames=max(1, _int("OCCUPY_FRAMES", 4)),
        free_frames=max(1, _int("FREE_FRAMES", 8)),
        realtime=_bool("CV_REALTIME", True),
        loop_video=_bool("CV_LOOP_VIDEO", False),
        reconnect_seconds=max(1.0, _float("CAMERA_RECONNECT_SECONDS", 5.0)),
        heartbeat_seconds=max(1.0, _float("CV_HEARTBEAT_SECONDS", 5.0)),
        save_overlay=_bool("CV_SAVE_OVERLAY", True),
        overlay_path=_path("CV_OVERLAY_PATH", "data/videos/overlay.mp4", REPO_ROOT),
        snapshot_dir=_path("SNAPSHOTS_DIR", "data/snapshots", REPO_ROOT),
        log_level=_env("CV_LOG_LEVEL", "INFO").upper(),
    )
