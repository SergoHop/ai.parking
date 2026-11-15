"""Helpers for marking spot polygons on a reference frame.

Used by ``scripts/annotate_polygons.py``: the GUI collects pixel clicks, this
module converts them into normalized polygons and builds a JSON payload that
can be pasted into ``data/lots/*.json`` or sent to the admin API.
"""

from __future__ import annotations

import json
from typing import Iterable, Sequence

Point = tuple[float, float]
Polygon = list[Point]

MAX_COORD = 1.0


def clamp01(value: float) -> float:
    """Clamp a coordinate into the normalized 0..1 range."""
    return max(0.0, min(MAX_COORD, value))


def normalize_points(
    points: Iterable[Sequence[float]],
    width: int,
    height: int,
    digits: int = 4,
) -> Polygon:
    """Convert pixel coordinates into rounded normalized 0..1 points."""
    if width <= 0 or height <= 0:
        raise ValueError("frame size must be positive")

    polygon: Polygon = []
    for x, y in points:
        polygon.append(
            (round(clamp01(float(x) / width), digits), round(clamp01(float(y) / height), digits))
        )
    return polygon


def rect_polygon(first: Sequence[float], second: Sequence[float]) -> list[Point]:
    """Build a rectangle polygon (pixels) from two opposite corners."""
    x1, y1 = float(first[0]), float(first[1])
    x2, y2 = float(second[0]), float(second[1])
    left, right = sorted((x1, x2))
    top, bottom = sorted((y1, y2))
    return [(left, top), (right, top), (right, bottom), (left, bottom)]


def validate_polygon(polygon: Sequence[Sequence[float]]) -> None:
    """Check a polygon against the same rules as the Go backend."""
    if len(polygon) < 3:
        raise ValueError("polygon must contain at least 3 points")
    for index, point in enumerate(polygon):
        if len(point) != 2:
            raise ValueError(f"polygon point {index} must have exactly 2 coordinates")
        x, y = float(point[0]), float(point[1])
        if not (0.0 <= x <= 1.0 and 0.0 <= y <= 1.0):
            raise ValueError(f"polygon point {index} must be normalized to 0..1")


def spot_code(prefix: str, index: int) -> str:
    """Return a spot code such as ``A-01`` for index 1."""
    return f"{prefix}-{index:02d}"


def build_payload(
    polygons: Sequence[Sequence[Sequence[float]]],
    prefix: str = "A",
    start: int = 1,
) -> dict:
    """Build ``{"spots": [{"code": ..., "polygon": ...}]}`` from polygons."""
    spots = []
    for offset, polygon in enumerate(polygons):
        validate_polygon(polygon)
        spots.append(
            {
                "code": spot_code(prefix, start + offset),
                "polygon": [[float(x), float(y)] for x, y in polygon],
            }
        )
    return {"spots": spots}


def payload_to_json(payload: dict) -> str:
    """Serialize an annotation payload with stable formatting."""
    return json.dumps(payload, ensure_ascii=False, indent=2) + "\n"
