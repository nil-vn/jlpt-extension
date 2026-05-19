#!/usr/bin/env python3
"""Generate ignored raster icon assets from the reviewed SVG source.

The repository keeps `build/assets/appicon.svg` as the source asset and does not
track generated PNG/ICO/ICNS/build outputs. Wails packaging still needs a PNG
input, so this script creates `build/generated/appicon.png` deterministically
without requiring ImageMagick or other native image tooling.
"""

from __future__ import annotations

import math
import struct
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SVG_SOURCE = ROOT / "build" / "assets" / "appicon.svg"
PNG_OUTPUT = ROOT / "build" / "generated" / "appicon.png"
SIZE = 1024


def _chunk(kind: bytes, data: bytes) -> bytes:
    return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data) & 0xFFFFFFFF)


def _lerp(a: int, b: int, t: float) -> int:
    return int(a * (1 - t) + b * t)


def _mix(c0: tuple[int, int, int], c1: tuple[int, int, int], t: float) -> tuple[int, int, int]:
    return tuple(_lerp(c0[index], c1[index], t) for index in range(3))


def _rounded_rect_contains(x: float, y: float, left: float, top: float, right: float, bottom: float, radius: float) -> bool:
    dx = max(left + radius - x, 0, x - (right - radius))
    dy = max(top + radius - y, 0, y - (bottom - radius))
    return dx * dx + dy * dy <= radius * radius


def _line_contains(x: float, y: float, x0: float, y0: float, x1: float, width: float) -> bool:
    return x0 <= x <= x1 and y0 - width / 2 <= y <= y0 + width / 2


def generate_png() -> None:
    if not SVG_SOURCE.exists():
        raise FileNotFoundError(f"Icon source not found: {SVG_SOURCE}")

    rows: list[bytes] = []
    orange = (249, 115, 22)
    red = (239, 68, 68)
    violet = (124, 58, 237)
    cream = (255, 247, 237)
    white = (255, 255, 255)
    ink = (17, 24, 39)

    for y in range(SIZE):
        row = bytearray()
        for x in range(SIZE):
            corner_radius = 224
            dx = max(corner_radius - x, 0, x - (SIZE - corner_radius - 1))
            dy = max(corner_radius - y, 0, y - (SIZE - corner_radius - 1))
            alpha = 255 if dx * dx + dy * dy <= corner_radius * corner_radius else 0

            t = (x + y) / (2 * (SIZE - 1))
            if t < 0.52:
                rgb = _mix(orange, red, t / 0.52)
            else:
                rgb = _mix(red, violet, (t - 0.52) / 0.48)

            cx, cy = x - 512, y - 512
            angle = -math.radians(8)
            rx = cx * math.cos(angle) - cy * math.sin(angle) + 512
            ry = cx * math.sin(angle) + cy * math.cos(angle) + 512
            if _rounded_rect_contains(rx, ry, 252, 180, 772, 844, 72):
                rgb = tuple(int(value * 0.82) for value in rgb)

            if _rounded_rect_contains(x, y, 228, 164, 748, 828, 72):
                u = (y - 164) / 664
                rgb = _mix(cream, white, u)

            for x0, y0, x1, width, colour in [
                (328, 292, 648, 42, orange),
                (328, 422, 568, 34, (252, 148, 84)),
                (328, 724, 648, 42, violet),
            ]:
                if _line_contains(x, y, x0, y0, x1, width):
                    rgb = colour

            # Stylised 日 glyph for the app icon center.
            if 414 <= x <= 610 and 478 <= y <= 624 and (x < 446 or x > 578 or y < 510 or y > 592 or 540 <= y <= 562):
                rgb = ink

            row += bytes((*rgb, alpha))
        rows.append(bytes([0]) + row)

    raw = b"".join(rows)
    png = (
        b"\x89PNG\r\n\x1a\n"
        + _chunk(b"IHDR", struct.pack(">IIBBBBB", SIZE, SIZE, 8, 6, 0, 0, 0))
        + _chunk(b"IDAT", zlib.compress(raw, 9))
        + _chunk(b"IEND", b"")
    )
    PNG_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    PNG_OUTPUT.write_bytes(png)
    print(f"Generated {PNG_OUTPUT.relative_to(ROOT)} from {SVG_SOURCE.relative_to(ROOT)}")


if __name__ == "__main__":
    generate_png()
