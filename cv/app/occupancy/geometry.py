"""Geometry helpers for normalized spot polygons and detected boxes."""

from __future__ import annotations

from typing import Any, Iterable, Sequence

Point = tuple[float, float]
Polygon = list[Point]
Box = tuple[float, float, float, float]


class PolygonError(ValueError):
    """Raised when a spot polygon cannot be parsed."""


def parse_polygon(raw: Any) -> Polygon:
    """Parse a JSON polygon [[x, y], ...] normalized to 0..1."""
    if not isinstance(raw, list) or len(raw) < 3:
        raise PolygonError("polygon must contain at least 3 points")

    polygon: Polygon = []
    for index, point in enumerate(raw):
        if not isinstance(point, (list, tuple)) or len(point) != 2:
            raise PolygonError(f"polygon point {index} must have two coordinates")
        x, y = float(point[0]), float(point[1])
        if not (0.0 <= x <= 1.0 and 0.0 <= y <= 1.0):
            raise PolygonError(f"polygon point {index} must be normalized to 0..1")
        polygon.append((x, y))
    return polygon


def point_in_polygon(point: Point, polygon: Sequence[Point]) -> bool:
    """Ray casting test; points exactly on an edge may go either way."""
    x, y = point
    inside = False
    count = len(polygon)
    for i in range(count):
        x1, y1 = polygon[i]
        x2, y2 = polygon[(i + 1) % count]
        if (y1 > y) != (y2 > y):
            intersect_x = (x2 - x1) * (y - y1) / (y2 - y1) + x1
            if x < intersect_x:
                inside = not inside
    return inside


def box_center(box: Box) -> Point:
    x1, y1, x2, y2 = box
    return ((x1 + x2) / 2.0, (y1 + y2) / 2.0)


def box_points(box: Box) -> list[Point]:
    x1, y1, x2, y2 = box
    return [
        ((x1 + x2) / 2.0, (y1 + y2) / 2.0),
        (x1, y1),
        (x2, y1),
        (x2, y2),
        (x1, y2),
    ]


def box_match_score(box: Box, polygon: Sequence[Point]) -> float:
    """Fraction of the box center/corners that fall inside the spot polygon."""
    points = box_points(box)
    hits = sum(1 for point in points if point_in_polygon(point, polygon))
    return hits / len(points)


def box_center_inside(box: Box, polygon: Sequence[Point]) -> bool:
    return point_in_polygon(box_center(box), polygon)


def normalize_box(box: Iterable[float], width: int, height: int) -> Box:
    x1, y1, x2, y2 = (float(value) for value in box)
    return (
        max(0.0, min(1.0, x1 / width)),
        max(0.0, min(1.0, y1 / height)),
        max(0.0, min(1.0, x2 / width)),
        max(0.0, min(1.0, y2 / height)),
    )
