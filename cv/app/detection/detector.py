"""Vehicle detection with Ultralytics YOLO."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import numpy as np

logger = logging.getLogger(__name__)

VEHICLE_CLASS_NAMES = {"car", "truck", "bus", "motorcycle", "vehicle"}


@dataclass(frozen=True)
class Detection:
    box: tuple[float, float, float, float]  # pixels, xyxy
    confidence: float
    class_id: int
    class_name: str


class VehicleDetector:
    def __init__(
        self,
        model_path: Path,
        confidence: float = 0.4,
        device: str = "auto",
    ) -> None:
        from ultralytics import YOLO  # heavy import, keep it lazy

        if not model_path.exists():
            raise FileNotFoundError(
                f"YOLO weights not found: {model_path}. "
                "Run `make cv-model` or put yolo11n.pt into cv/models/."
            )

        self.model = YOLO(str(model_path))
        self.confidence = confidence
        self.device = self._resolve_device(device)
        self.class_ids = self._vehicle_class_ids(self.model.names)
        logger.info(
            "detector ready: model=%s device=%s classes=%s",
            model_path.name,
            self.device,
            self.class_ids,
        )

    def detect(self, frame: np.ndarray) -> list[Detection]:
        try:
            return self._detect(frame)
        except Exception as exc:  # noqa: BLE001 - fall back when MPS/CUDA fails
            if self.device == "cpu":
                raise
            logger.warning("inference failed on %s (%s), falling back to CPU", self.device, exc)
            self.device = "cpu"
            return self._detect(frame)

    def _detect(self, frame: np.ndarray) -> list[Detection]:
        # Ultralytics accepts numpy arrays in BGR order, exactly as OpenCV gives them.
        kwargs: dict[str, Any] = {
            "source": frame,
            "conf": self.confidence,
            "device": self.device,
            "verbose": False,
        }
        if self.class_ids:
            kwargs["classes"] = self.class_ids

        result = self.model.predict(**kwargs)[0]
        detections: list[Detection] = []
        if result.boxes is None:
            return detections

        boxes = result.boxes
        xyxy = boxes.xyxy.cpu().numpy()
        confidences = boxes.conf.cpu().numpy()
        classes = boxes.cls.cpu().numpy().astype(int)
        names = result.names or self.model.names

        for coordinates, confidence, class_id in zip(xyxy, confidences, classes):
            detections.append(
                Detection(
                    box=(
                        float(coordinates[0]),
                        float(coordinates[1]),
                        float(coordinates[2]),
                        float(coordinates[3]),
                    ),
                    confidence=float(confidence),
                    class_id=int(class_id),
                    class_name=self._class_name(names, int(class_id)),
                )
            )
        return detections

    @staticmethod
    def _class_name(names: Any, class_id: int) -> str:
        if isinstance(names, dict):
            return str(names.get(class_id, class_id))
        try:
            return str(names[class_id])
        except (IndexError, KeyError, TypeError):
            return str(class_id)

    @staticmethod
    def _vehicle_class_ids(names: Any) -> list[int]:
        if not isinstance(names, dict):
            return []
        ids = [
            class_id
            for class_id, name in names.items()
            if str(name).lower() in VEHICLE_CLASS_NAMES
        ]
        return sorted(ids)

    @staticmethod
    def _resolve_device(device: str) -> str:
        """Pick an inference device.

        ``auto`` prefers CUDA and falls back to CPU. Apple MPS is never chosen
        automatically: some torch/YOLO combinations abort the process with an
        uncatchable Objective-C exception, so it has to be requested explicitly
        with ``CV_DEVICE=mps``.
        """
        if device and device != "auto":
            return device
        try:
            import torch

            if torch.cuda.is_available():
                return "0"
        except Exception:  # pragma: no cover - torch is always present with ultralytics
            pass
        return "cpu"
