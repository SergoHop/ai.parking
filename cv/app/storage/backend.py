"""HTTP client for the Go internal API."""

from __future__ import annotations

import logging
from datetime import datetime

import httpx

from ..schemas.events import CameraHeartbeatPayload, SpotEventPayload, SpotEventsPayload

logger = logging.getLogger(__name__)


class BackendError(RuntimeError):
    pass


class BackendClient:
    def __init__(self, base_url: str, internal_token: str, timeout: float = 5.0) -> None:
        self._client = httpx.Client(
            base_url=base_url.rstrip("/"),
            headers={"X-Internal-Token": internal_token},
            timeout=timeout,
        )

    def close(self) -> None:
        self._client.close()

    def get_spots(self, lot_id: int) -> list[dict]:
        response = self._client.get(f"/api/parking-lots/{lot_id}/spots")
        if response.status_code != 200:
            raise BackendError(f"GET spots failed: HTTP {response.status_code}")
        return response.json()

    def send_heartbeat(
        self,
        camera_id: int,
        status: str,
        last_frame_at: datetime | None = None,
        fps: int | None = None,
        resolution: str | None = None,
    ) -> None:
        payload = CameraHeartbeatPayload(
            status=status,  # type: ignore[arg-type]
            last_frame_at=last_frame_at,
            fps=fps,
            resolution=resolution,
        )
        response = self._client.post(
            f"/api/internal/cameras/{camera_id}/heartbeat",
            json=payload.model_dump(mode="json", exclude_none=True),
        )
        if response.status_code != 204:
            raise BackendError(f"heartbeat failed: HTTP {response.status_code} {response.text}")

    def send_events(self, camera_id: int, events: list[SpotEventPayload]) -> None:
        if not events:
            return
        payload = SpotEventsPayload(camera_id=camera_id, events=events)
        response = self._client.post(
            "/api/internal/events",
            json=payload.model_dump(mode="json", exclude_none=True),
        )
        if response.status_code != 200:
            raise BackendError(f"events failed: HTTP {response.status_code} {response.text}")
