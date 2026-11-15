"""Video sources: file, webcam and RTSP with reconnect support."""

from __future__ import annotations

import logging
import time
from pathlib import Path

import cv2
import numpy as np

from ..core.config import REPO_ROOT

logger = logging.getLogger(__name__)


class SourceError(RuntimeError):
    pass


class VideoSource:
    def __init__(self, kind: str, source: str, capture: cv2.VideoCapture) -> None:
        self.kind = kind
        self.source = source
        self.capture = capture
        self.fps = float(capture.get(cv2.CAP_PROP_FPS) or 0.0)
        self.width = int(capture.get(cv2.CAP_PROP_FRAME_WIDTH) or 0)
        self.height = int(capture.get(cv2.CAP_PROP_FRAME_HEIGHT) or 0)

    @property
    def is_file(self) -> bool:
        return self.kind == "file"

    @classmethod
    def open(cls, source_type: str, source: str) -> "VideoSource":
        kind = resolve_source_type(source_type, source)
        target: str | int
        display_source = source
        if kind == "webcam":
            target = int(source)
        elif kind == "file":
            path = Path(source).expanduser()
            if not path.is_absolute():
                path = (REPO_ROOT / path).resolve()
            if not path.exists():
                raise SourceError(f"video file not found: {path}")
            target = str(path)
            display_source = target
        else:
            target = source

        capture = cv2.VideoCapture(target)
        if kind == "rtsp":
            capture.set(cv2.CAP_PROP_BUFFERSIZE, 1)
        if not capture.isOpened():
            capture.release()
            raise SourceError(f"cannot open {kind} source: {source}")

        instance = cls(kind=kind, source=display_source, capture=capture)
        logger.info(
            "source opened: kind=%s fps=%.1f size=%dx%d",
            kind,
            instance.fps,
            instance.width,
            instance.height,
        )
        return instance

    def read(self) -> np.ndarray | None:
        ok, frame = self.capture.read()
        if not ok or frame is None:
            return None
        return frame

    def reopen(self) -> None:
        self.capture.release()
        time.sleep(0.2)
        capture = cv2.VideoCapture(int(self.source) if self.kind == "webcam" else self.source)
        if self.kind == "rtsp":
            capture.set(cv2.CAP_PROP_BUFFERSIZE, 1)
        if not capture.isOpened():
            capture.release()
            raise SourceError(f"cannot reopen {self.kind} source: {self.source}")
        self.capture = capture

    def release(self) -> None:
        self.capture.release()

    def resolution(self) -> str | None:
        if self.width and self.height:
            return f"{self.width}x{self.height}"
        return None


def resolve_source_type(source_type: str, source: str) -> str:
    if source_type and source_type != "auto":
        if source_type not in {"file", "webcam", "rtsp"}:
            raise SourceError(f"unknown source type: {source_type}")
        return source_type

    lowered = source.lower()
    if lowered.startswith(("rtsp://", "rtmp://", "http://", "https://")):
        return "rtsp"
    if source.isdigit():
        return "webcam"
    return "file"
