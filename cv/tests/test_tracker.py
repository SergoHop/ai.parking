from __future__ import annotations

from datetime import datetime, timezone

import pytest

from app.occupancy.tracker import OccupancyTracker, Spot

SPOTS = [
    Spot(id=1, code="A-01", polygon=[(0.1, 0.1), (0.2, 0.1), (0.2, 0.2), (0.1, 0.2)]),
    Spot(id=2, code="A-02", polygon=[(0.3, 0.1), (0.4, 0.1), (0.4, 0.2), (0.3, 0.2)]),
]

CAR_IN_FIRST = ((0.12, 0.12, 0.18, 0.18), 0.9)
EMPTY: list[tuple[tuple[float, float, float, float], float]] = []


def make_tracker(occupy: int = 3, free: int = 5) -> OccupancyTracker:
    return OccupancyTracker(SPOTS, occupy_frames=occupy, free_frames=free)


def test_free_after_consecutive_empty_frames() -> None:
    tracker = make_tracker()
    now = datetime.now(timezone.utc)

    for _ in range(4):
        assert tracker.update(EMPTY, at=now) == []

    changes = tracker.update(EMPTY, at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "free"), (2, "free")]
    assert tracker.statuses() == {1: "free", 2: "free"}


def test_occupied_after_consecutive_detections() -> None:
    tracker = make_tracker()
    now = datetime.now(timezone.utc)

    for _ in range(2):
        assert tracker.update([CAR_IN_FIRST], at=now) == []

    changes = tracker.update([CAR_IN_FIRST], at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "occupied")]
    assert changes[0].confidence == pytest.approx(0.9)
    assert tracker.statuses()[1] == "occupied"
    assert tracker.statuses()[2] == "unknown"


def test_flicker_resets_counter() -> None:
    tracker = make_tracker(occupy=3, free=5)
    now = datetime.now(timezone.utc)

    tracker.update([CAR_IN_FIRST], at=now)
    tracker.update([CAR_IN_FIRST], at=now)
    tracker.update(EMPTY, at=now)  # one empty frame resets the hit counter

    tracker.update([CAR_IN_FIRST], at=now)
    tracker.update([CAR_IN_FIRST], at=now)
    assert tracker.statuses()[1] == "unknown"

    changes = tracker.update([CAR_IN_FIRST], at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "occupied")]


def test_free_after_occupation() -> None:
    tracker = make_tracker(occupy=3, free=2)
    now = datetime.now(timezone.utc)

    for _ in range(3):
        tracker.update([CAR_IN_FIRST], at=now)
    assert tracker.statuses()[1] == "occupied"

    tracker.update(EMPTY, at=now)
    changes = tracker.update(EMPTY, at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "free")]


def test_best_confidence_wins() -> None:
    tracker = make_tracker(occupy=1)
    now = datetime.now(timezone.utc)

    changes = tracker.update(
        [
            ((0.12, 0.12, 0.18, 0.18), 0.55),
            ((0.13, 0.13, 0.19, 0.19), 0.95),
        ],
        at=now,
    )
    assert changes[0].confidence == pytest.approx(0.95)


def test_detection_in_other_spot_does_not_match() -> None:
    tracker = make_tracker(occupy=1)
    now = datetime.now(timezone.utc)
    car_in_second = ((0.32, 0.12, 0.38, 0.18), 0.8)

    changes = tracker.update([car_in_second], at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(2, "occupied")]


def test_mark_unknown() -> None:
    tracker = make_tracker(occupy=1)
    now = datetime.now(timezone.utc)
    tracker.update([CAR_IN_FIRST], at=now)

    changes = tracker.mark_unknown(at=now)
    assert {c.status for c in changes} == {"unknown"}
    assert tracker.statuses() == {1: "unknown", 2: "unknown"}


def test_initial_status_prevents_duplicate_events() -> None:
    spots = [
        Spot(id=1, code="A-01", polygon=SPOTS[0].polygon, initial_status="occupied"),
        Spot(id=2, code="A-02", polygon=SPOTS[1].polygon, initial_status="free"),
    ]
    tracker = OccupancyTracker(spots, occupy_frames=2, free_frames=2)
    now = datetime.now(timezone.utc)

    assert tracker.statuses() == {1: "occupied", 2: "free"}
    assert tracker.update([CAR_IN_FIRST], at=now) == []
    assert tracker.update([CAR_IN_FIRST], at=now) == []


def test_initial_occupied_becomes_free_when_spot_is_empty() -> None:
    spots = [Spot(id=1, code="A-01", polygon=SPOTS[0].polygon, initial_status="occupied")]
    tracker = OccupancyTracker(spots, occupy_frames=2, free_frames=2)
    now = datetime.now(timezone.utc)

    assert tracker.update([], at=now) == []
    changes = tracker.update([], at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "free")]


def test_weak_detection_keeps_occupied_spot() -> None:
    spots = [Spot(id=1, code="A-01", polygon=SPOTS[0].polygon, initial_status="occupied")]
    tracker = OccupancyTracker(spots, occupy_frames=2, free_frames=2)
    now = datetime.now(timezone.utc)
    weak_car = ((0.12, 0.12, 0.18, 0.18), 0.27)

    for _ in range(10):
        assert tracker.update([weak_car], at=now) == []

    assert tracker.statuses() == {1: "occupied"}


def test_weak_detection_does_not_occupy_free_spot() -> None:
    tracker = make_tracker(occupy=2, free=2)
    now = datetime.now(timezone.utc)
    weak_car = ((0.12, 0.12, 0.18, 0.18), 0.25)

    for _ in range(5):
        tracker.update([weak_car], at=now)

    assert tracker.statuses()[1] == "unknown"


def test_weak_detection_does_not_block_free_after_strong_gone() -> None:
    spots = [Spot(id=1, code="A-01", polygon=SPOTS[0].polygon, initial_status="occupied")]
    tracker = OccupancyTracker(spots, occupy_frames=2, free_frames=2)
    now = datetime.now(timezone.utc)
    weak_car = ((0.12, 0.12, 0.18, 0.18), 0.25)

    assert tracker.update([weak_car], at=now) == []
    assert tracker.update([], at=now) == []
    changes = tracker.update([], at=now)
    assert [(c.spot_id, c.status) for c in changes] == [(1, "free")]
