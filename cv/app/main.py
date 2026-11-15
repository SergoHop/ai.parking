"""FastAPI entrypoint for the CV service."""

from __future__ import annotations

import logging
from contextlib import asynccontextmanager
from typing import AsyncIterator

from fastapi import FastAPI
from fastapi.responses import JSONResponse

from .core.config import Settings, load_settings
from .pipeline.worker import CameraWorker
from .storage.backend import BackendClient

logger = logging.getLogger(__name__)


def _configure_logging(settings: Settings) -> None:
    logging.basicConfig(
        level=getattr(logging, settings.log_level, logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings = load_settings()
    _configure_logging(settings)

    backend = None
    if settings.backend_url and settings.internal_token:
        backend = BackendClient(settings.backend_url, settings.internal_token)

    worker = CameraWorker(settings, backend)
    app.state.settings = settings
    app.state.worker = worker

    if settings.can_start:
        logger.info(
            "starting CV worker: camera_id=%s lot_id=%s source=%s",
            settings.camera_id,
            settings.lot_id,
            settings.source,
        )
        worker.start()
    else:
        logger.info(
            "CV worker is idle: set CV_SOURCE, CV_CAMERA_ID, CV_LOT_ID and INTERNAL_TOKEN"
        )

    try:
        yield
    finally:
        worker.stop()


app = FastAPI(title="AI Parking CV", version="0.1.0", lifespan=lifespan)


@app.get("/health")
def health() -> JSONResponse:
    worker: CameraWorker = app.state.worker
    snapshot = worker.snapshot()
    status = snapshot["status"]
    if status == "error":
        return JSONResponse(status_code=503, content={"status": "error", "worker": snapshot})
    if status == "idle":
        return JSONResponse(status_code=200, content={"status": "idle", "worker": snapshot})
    return JSONResponse(status_code=200, content={"status": "ok", "worker": snapshot})


@app.get("/status")
def status() -> dict:
    worker: CameraWorker = app.state.worker
    return worker.snapshot()
