from __future__ import annotations

from app.capture.source import resolve_source_type


def test_resolve_source_type() -> None:
    assert resolve_source_type("auto", "rtsp://camera/stream") == "rtsp"
    assert resolve_source_type("auto", "http://example.com/video.mp4") == "rtsp"
    assert resolve_source_type("auto", "0") == "webcam"
    assert resolve_source_type("auto", "../data/videos/test.mp4") == "file"
    assert resolve_source_type("file", "whatever") == "file"
