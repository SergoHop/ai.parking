"""Per-spot occupancy state with frame hysteresis."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable, Sequence

from .geometry import Box, Polygon, box_center_inside, box_match_score


@dataclass(frozen=True)
class Spot:
    id: int
    code: str
    polygon: Polygon
    initial_status: str = "unknown"


@dataclass
class SpotState:
    spot: Spot
    status: str = "unknown"
    confidence: float | None = None
    hit_count: int = 0
    miss_count: int = 0


@dataclass(frozen=True)
class StatusChange:
    spot_id: int
    code: str
    status: str
    confidence: float | None
    captured_at: datetime


@dataclass
class OccupancyTracker:
    """Turns noisy per-frame detections into stable spot states.

    A spot becomes occupied after ``occupy_frames`` consecutive frames with a
    matching vehicle, and free after ``free_frames`` consecutive empty frames.

    Detection evidence is asymmetric: only detections at or above
    ``occupied_confidence`` can flip a spot to occupied, while any detection
    at or above ``weak_confidence`` blocks the transition to free. This keeps
    a parked car marked occupied while a passing vehicle partly hides it.
    """

    spots: Sequence[Spot]
    occupy_frames: int = 4
    free_frames: int = 8
    match_threshold: float = 0.6
    occupied_confidence: float = 0.4
    weak_confidence: float = 0.2
    states: dict[int, SpotState] = field(init=False)

    def __post_init__(self) -> None:
        self.states = {
            spot.id: SpotState(
                spot=spot,
                status=spot.initial_status if spot.initial_status in {"free", "occupied"} else "unknown",
            )
            for spot in self.spots
        }

    def update(
        self,
        detections: Iterable[tuple[Box, float]],
        at: datetime | None = None,
    ) -> list[StatusChange]:
        captured_at = at or datetime.now(timezone.utc)
        detections = list(detections)
        changes: list[StatusChange] = []

        for state in self.states.values():
            best_confidence = self._match_confidence(state.spot.polygon, detections)

            if best_confidence is not None and best_confidence >= self.occupied_confidence:
                # Confident detection: this is normal "the spot is taken" evidence.
                state.hit_count += 1
                state.miss_count = 0
                state.confidence = best_confidence

                if state.status != "occupied" and state.hit_count >= self.occupy_frames:
                    state.status = "occupied"
                    changes.append(self._change(state, captured_at))
            elif best_confidence is not None:
                # Weak detection (occlusion, blur, small box). It is not enough
                # to occupy a spot, but it must not free an occupied one either.
                state.miss_count = 0
                state.hit_count = 0
                if state.status == "occupied":
                    state.confidence = best_confidence
            else:
                state.miss_count += 1
                state.hit_count = 0

                if state.status != "free" and state.miss_count >= self.free_frames:
                    state.status = "free"
                    state.confidence = None
                    changes.append(self._change(state, captured_at))

        return changes

    def mark_unknown(self, at: datetime | None = None) -> list[StatusChange]:
        captured_at = at or datetime.now(timezone.utc)
        changes: list[StatusChange] = []
        for state in self.states.values():
            state.hit_count = 0
            state.miss_count = 0
            if state.status != "unknown":
                state.status = "unknown"
                state.confidence = None
                changes.append(self._change(state, captured_at))
        return changes

    def statuses(self) -> dict[int, str]:
        return {spot_id: state.status for spot_id, state in self.states.items()}

    def _match_confidence(
        self,
        polygon: Polygon,
        detections: Sequence[tuple[Box, float]],
    ) -> float | None:
        best: float | None = None
        for box, confidence in detections:
            if box_center_inside(box, polygon) or box_match_score(box, polygon) >= self.match_threshold:
                if best is None or confidence > best:
                    best = confidence
        return best

    @staticmethod
    def _change(state: SpotState, captured_at: datetime) -> StatusChange:
        return StatusChange(
            spot_id=state.spot.id,
            code=state.spot.code,
            status=state.status,
            confidence=state.confidence,
            captured_at=captured_at,
        )
