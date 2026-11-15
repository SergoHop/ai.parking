"""Run one camera worker without FastAPI (useful for debugging)."""

from __future__ import annotations

import logging
import sys

from ..core.config import load_settings
from ..pipeline.worker import CameraWorker
from ..storage.backend import BackendClient


def main() -> None:
    settings = load_settings()
    logging.basicConfig(
        level=getattr(logging, settings.log_level, logging.INFO),
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    if not settings.can_start:
        print(
            "CV worker is not configured.\n"
            "Set CV_SOURCE, CV_CAMERA_ID, CV_LOT_ID and INTERNAL_TOKEN in .env.",
            file=sys.stderr,
        )
        raise SystemExit(2)

    backend = BackendClient(settings.backend_url, settings.internal_token)
    worker = CameraWorker(settings, backend)
    try:
        worker.run()
    except KeyboardInterrupt:
        worker.stop()


if __name__ == "__main__":
    main()
