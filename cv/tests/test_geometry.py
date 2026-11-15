from __future__ import annotations

import pytest

from app.occupancy.geometry import (
    PolygonError,
    box_center,
    box_center_inside,
    box_match_score,
    normalize_box,
    parse_polygon,
    point_in_polygon,
)

SQUARE = [(0.2, 0.2), (0.4, 0.2), (0.4, 0.4), (0.2, 0.4)]


def test_parse_polygon_ok() -> None:
    assert parse_polygon([[0.1, 0.1], [0.2, 0.1], [0.2, 0.2]]) == [
        (0.1, 0.1),
        (0.2, 0.1),
        (0.2, 0.2),
    ]


@pytest.mark.parametrize(
    "raw",
    [
        [[0.1, 0.1], [0.2, 0.2]],
        [[0.1, 0.1], [1.2, 0.2], [0.2, 0.3]],
        [[0.1, 0.1, 0.0], [0.2, 0.2], [0.3, 0.3]],
        "not a list",
    ],
)
def test_parse_polygon_invalid(raw: object) -> None:
    with pytest.raises(PolygonError):
        parse_polygon(raw)


def test_point_in_polygon() -> None:
    assert point_in_polygon((0.3, 0.3), SQUARE)
    assert not point_in_polygon((0.1, 0.1), SQUARE)
    assert not point_in_polygon((0.5, 0.3), SQUARE)


def test_box_center_and_match() -> None:
    inside = (0.25, 0.25, 0.35, 0.35)
    outside = (0.6, 0.6, 0.7, 0.7)

    assert box_center(inside) == pytest.approx((0.3, 0.3))
    assert box_center_inside(inside, SQUARE)
    assert not box_center_inside(outside, SQUARE)
    assert box_match_score(inside, SQUARE) == pytest.approx(1.0)
    assert box_match_score(outside, SQUARE) == pytest.approx(0.0)


def test_box_partially_covering_spot() -> None:
    # Center is inside, two corners are outside.
    box = (0.15, 0.25, 0.35, 0.35)
    assert box_center_inside(box, SQUARE)
    assert box_match_score(box, SQUARE) == pytest.approx(0.6)


def test_normalize_box() -> None:
    assert normalize_box((0, 0, 640, 360), 1280, 720) == (0.0, 0.0, 0.5, 0.5)
    assert normalize_box((-10, -10, 2000, 1000), 1280, 720) == (0.0, 0.0, 1.0, 1.0)
