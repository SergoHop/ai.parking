#!/usr/bin/env python3
"""Mark parking spot polygons on a reference frame with the mouse.

Usage (from the repository root, with the CV virtualenv):

    cv/.venv/bin/python scripts/annotate_polygons.py \
        --frame data/frames/reference.jpg \
        --output data/lots/spots.json

    # or grab a frame from a video at 12.5 s
    cv/.venv/bin/python scripts/annotate_polygons.py \
        --video data/videos/pocket.mp4 --time 12.5 \
        --output data/lots/spots.json

Controls:
    left click   add a polygon corner (the spot is finished after N clicks)
    right click  remove the last corner of the current spot
    u            undo the last corner
    r            reset the current spot
    d            delete the last finished spot
    s            save JSON
    g            toggle the 10% grid
    q / Esc      quit

The JSON is compatible with the admin API and ``data/lots/*.json``; map
coordinates still have to be filled in separately.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT / "cv"))

import cv2  # noqa: E402
import numpy as np  # noqa: E402

from app.annotation import (  # noqa: E402
    build_payload,
    normalize_points,
    payload_to_json,
    rect_polygon,
    spot_code,
)

WINDOW = "parking spots"
INSTRUCTIONS = (
    "LMB: add point   RMB: undo point   u: undo   r: reset   "
    "d: delete last   s: save   g: grid   q: quit"
)
GREEN = (80, 220, 80)
YELLOW = (0, 220, 255)
GRAY = (160, 160, 160)
WHITE = (255, 255, 255)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Mark spot polygons on a reference frame.")
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--frame", type=Path, help="reference frame image (jpg/png)")
    source.add_argument("--video", type=Path, help="video file to grab a frame from")

    parser.add_argument("--time", type=float, default=0.0, help="seconds into --video")
    parser.add_argument("--output", type=Path, default=Path("data/lots/spots.json"))
    parser.add_argument("--prefix", default="A", help="spot code prefix (default: A)")
    parser.add_argument("--start", type=int, default=1, help="first spot number")
    parser.add_argument(
        "--mode",
        choices=("poly", "rect"),
        default="poly",
        help="poly: click every corner; rect: click two opposite corners",
    )
    parser.add_argument("--points", type=int, default=4, help="corners per spot in poly mode")
    parser.add_argument("--max-width", type=int, default=1600, help="window width limit")
    return parser.parse_args(argv)


def load_frame(args: argparse.Namespace):
    if args.frame:
        frame = cv2.imread(str(args.frame))
        if frame is None:
            raise SystemExit(f"cannot read frame: {args.frame}")
        return frame

    capture = cv2.VideoCapture(str(args.video))
    if not capture.isOpened():
        raise SystemExit(f"cannot open video: {args.video}")
    capture.set(cv2.CAP_PROP_POS_MSEC, max(0.0, args.time) * 1000.0)
    ok, frame = capture.read()
    capture.release()
    if not ok:
        raise SystemExit(f"cannot read a frame at {args.time:.3f}s from {args.video}")
    return frame


class Annotator:
    def __init__(self, frame, args: argparse.Namespace) -> None:
        self.frame = frame
        self.height, self.width = frame.shape[:2]
        self.args = args
        self.polygons: list[list[tuple[float, float]]] = []
        self.current: list[tuple[float, float]] = []
        self.show_grid = True
        self.saved = False

        max_width = max(320, args.max_width)
        self.scale = min(1.0, max_width / self.width)
        self.display = (
            cv2.resize(frame, (int(self.width * self.scale), int(self.height * self.scale)))
            if self.scale < 1.0
            else frame.copy()
        )

    # ------------------------------------------------------------------ input
    def on_mouse(self, event: int, x: int, y: int, flags: int, param) -> None:
        if event not in (cv2.EVENT_LBUTTONDOWN, cv2.EVENT_RBUTTONDOWN):
            return

        if event == cv2.EVENT_RBUTTONDOWN:
            self.undo_point()
            return

        point = (
            min(self.width - 1, max(0, int(round(x / self.scale)))),
            min(self.height - 1, max(0, int(round(y / self.scale)))),
        )
        self.current.append(point)
        if len(self.current) >= self.points_needed:
            self.finish_spot()

    def on_key(self, key: int) -> bool:
        """Handle a key press; return False when the window should close."""
        if key in (27, ord("q")):
            if self.polygons and not self.saved:
                print("warning: polygons are not saved (press s before q)", file=sys.stderr)
            return False
        if key == ord("u"):
            self.undo_point()
        elif key == ord("r"):
            self.current.clear()
        elif key == ord("d"):
            if self.polygons:
                removed = self.polygons.pop()
                self.saved = False
                print(f"deleted {spot_code(self.args.prefix, self.args.start + len(self.polygons))} "
                      f"({len(removed)} points)")
        elif key == ord("s"):
            self.save()
        elif key == ord("g"):
            self.show_grid = not self.show_grid
        return True

    # ----------------------------------------------------------------- state
    @property
    def points_needed(self) -> int:
        return 2 if self.args.mode == "rect" else max(3, self.args.points)

    def undo_point(self) -> None:
        if self.current:
            self.current.pop()

    def finish_spot(self) -> None:
        points = (
            rect_polygon(self.current[0], self.current[1])
            if self.args.mode == "rect"
            else list(self.current)
        )
        polygon = normalize_points(points, self.width, self.height)
        self.polygons.append(polygon)
        code = spot_code(self.args.prefix, self.args.start + len(self.polygons) - 1)
        print(f"{code}: {polygon}")
        self.current.clear()
        self.saved = False

    def save(self) -> None:
        payload = build_payload(self.polygons, prefix=self.args.prefix, start=self.args.start)
        self.args.output.parent.mkdir(parents=True, exist_ok=True)
        text = payload_to_json(payload)
        self.args.output.write_text(text, encoding="utf-8")
        self.saved = True
        print(f"saved {len(self.polygons)} spots to {self.args.output}")

    # ---------------------------------------------------------------- drawing
    def draw(self, canvas) -> None:
        if self.show_grid:
            for step in range(1, 10):
                x = int(canvas.shape[1] * step / 10)
                y = int(canvas.shape[0] * step / 10)
                cv2.line(canvas, (x, 0), (x, canvas.shape[0]), GRAY, 1)
                cv2.line(canvas, (0, y), (canvas.shape[1], y), GRAY, 1)

        for index, polygon in enumerate(self.polygons):
            points = [
                (int(x * canvas.shape[1]), int(y * canvas.shape[0])) for x, y in polygon
            ]
            cv2.polylines(canvas, [np.array(points, dtype="int32")], True, GREEN, 2)
            code = spot_code(self.args.prefix, self.args.start + index)
            cv2.putText(canvas, code, (points[0][0], max(16, points[0][1] - 6)),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.6, GREEN, 2, cv2.LINE_AA)

        if self.current:
            points = [
                (int(x * canvas.shape[1]), int(y * canvas.shape[0])) for x, y in self.current
            ]
            for point in points:
                cv2.circle(canvas, point, 4, YELLOW, -1)
            for start, end in zip(points, points[1:]):
                cv2.line(canvas, start, end, YELLOW, 2)

        header = (
            f"{INSTRUCTIONS}    spots={len(self.polygons)}  "
            f"need={self.points_needed} clicks  mode={self.args.mode}"
        )
        cv2.rectangle(canvas, (0, 0), (canvas.shape[1], 26), (20, 20, 20), -1)
        cv2.putText(canvas, header, (6, 18), cv2.FONT_HERSHEY_SIMPLEX, 0.45, WHITE, 1, cv2.LINE_AA)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    frame = load_frame(args)
    annotator = Annotator(frame, args)

    try:
        cv2.namedWindow(WINDOW, cv2.WINDOW_AUTOSIZE)
        cv2.setMouseCallback(WINDOW, annotator.on_mouse)
    except cv2.error as exc:
        print(
            "cannot open a GUI window (no desktop session?): "
            f"{exc}\nRun this script on a machine with a display.",
            file=sys.stderr,
        )
        return 1

    while True:
        canvas = annotator.display.copy()
        annotator.draw(canvas)
        cv2.imshow(WINDOW, canvas)
        key = cv2.waitKey(20) & 0xFF
        if key != 255 and not annotator.on_key(key):
            break

    cv2.destroyAllWindows()
    if annotator.polygons and not annotator.saved:
        print("polygons are not saved; rerun and press s", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
