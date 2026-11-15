from __future__ import annotations

import json

import pytest

from app.annotation import (
    build_payload,
    normalize_points,
    payload_to_json,
    rect_polygon,
    spot_code,
    validate_polygon,
)


def test_normalize_points_rounds_and_clamps() -> None:
    polygon = normalize_points([(640, 360), (2000, -20)], width=1280, height=720)
    assert polygon == [(0.5, 0.5), (1.0, 0.0)]


def test_normalize_points_rejects_bad_size() -> None:
    with pytest.raises(ValueError):
        normalize_points([(10, 10)], width=0, height=720)


def test_rect_polygon_orders_corners() -> None:
    assert rect_polygon((300, 200), (100, 50)) == [
        (100, 50),
        (300, 50),
        (300, 200),
        (100, 200),
    ]


def test_validate_polygon_accepts_quad() -> None:
    validate_polygon([(0.1, 0.1), (0.2, 0.1), (0.2, 0.2), (0.1, 0.2)])


def test_validate_polygon_rejects_short_and_out_of_range() -> None:
    with pytest.raises(ValueError):
        validate_polygon([(0.1, 0.1), (0.2, 0.1)])
    with pytest.raises(ValueError):
        validate_polygon([(0.1, 0.1), (1.2, 0.1), (0.2, 0.2)])


def test_spot_code_is_padded() -> None:
    assert spot_code("A", 1) == "A-01"
    assert spot_code("T", 12) == "T-12"


def test_build_payload_numbers_spots() -> None:
    polygon = [(0.1, 0.1), (0.2, 0.1), (0.2, 0.2), (0.1, 0.2)]
    payload = build_payload([polygon, polygon], prefix="P", start=3)

    assert [spot["code"] for spot in payload["spots"]] == ["P-03", "P-04"]
    assert payload["spots"][0]["polygon"] == [[0.1, 0.1], [0.2, 0.1], [0.2, 0.2], [0.1, 0.2]]


def test_payload_to_json_round_trips() -> None:
    polygon = [(0.1, 0.1), (0.2, 0.1), (0.2, 0.2)]
    text = payload_to_json(build_payload([polygon]))

    assert text.endswith("\n")
    assert json.loads(text)["spots"][0]["code"] == "A-01"
