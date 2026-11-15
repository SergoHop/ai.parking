"""Pydantic schemas for the Go internal API."""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel, Field

SpotStatus = Literal["free", "occupied", "unknown"]
CameraStatus = Literal["online", "offline", "error"]


class SpotEventPayload(BaseModel):
    spot_id: int
    status: SpotStatus
    confidence: float | None = Field(default=None, ge=0, le=1)
    captured_at: datetime
    snapshot_path: str | None = None


class SpotEventsPayload(BaseModel):
    camera_id: int | None = None
    events: list[SpotEventPayload]


class CameraHeartbeatPayload(BaseModel):
    status: CameraStatus
    last_frame_at: datetime | None = None
    fps: int | None = Field(default=None, gt=0)
    resolution: str | None = None
