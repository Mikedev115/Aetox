"""Renders appicon.png and windows/icon.ico from appicon-source.png: the square
picture as given, corners rounded, a faint light rim along the top edge and a
soft shadow underneath. The 16-32 frames skip the shadow -- at that size it is
mud, and the 10% of the tile the shadow needs is real pixels on a taskbar.

    pip install pillow
    python desktop/build/make-icon.py

Then run windows/msix/make-logos.ps1 so the Store set follows.
"""
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFilter

HERE = Path(__file__).resolve().parent
SRC = HERE / "appicon-source.png"
OUT_PNG = HERE / "appicon.png"
OUT_ICO = HERE / "windows" / "icon.ico"
RADIUS = 0.22   # corner, same as the previous icon
TILE = 0.90     # tile width as a share of the canvas; the rest is room for the shadow
MASTER = 1024

src = Image.open(SRC).convert("RGBA")
assert src.width == src.height


def rounded_mask(size, radius_frac=RADIUS, inset=0):
    big = size * 4
    m = Image.new("L", (big, big), 0)
    i = inset * 4
    ImageDraw.Draw(m).rounded_rectangle((i, i, big - 1 - i, big - 1 - i),
                                        radius=int((big - 2 * i) * radius_frac), fill=255)
    return m.resize((size, size), Image.LANCZOS)


def tile(size):
    """The picture, corner-rounded, with a faint light rim like a raised card."""
    im = src.resize((size, size), Image.LANCZOS)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(im, (0, 0), rounded_mask(size))
    # rim: a thin ring (outer mask minus inset mask), brighter at the top,
    # fading to nothing at the bottom -- light from above.
    w = max(1, round(size * 0.004))
    ring = ImageChops.subtract(rounded_mask(size), rounded_mask(size, inset=w))
    grad = Image.linear_gradient("L").resize((size, size)).point(lambda v: 255 - v)  # 255 top -> 0 bottom
    ring = ImageChops.multiply(ring, grad).point(lambda v: v * 0.32)
    out.paste(Image.new("RGBA", (size, size), (255, 255, 255, 255)), (0, 0), ring)
    return out


def with_shadow(size):
    """Tile shrunk into a transparent canvas with a soft shadow below it."""
    t = round(size * TILE)
    x = (size - t) // 2
    y = (size - t) // 2 - round(size * 0.012)
    canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    sh = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    sh.paste(Image.new("RGBA", (t, t), (0, 0, 0, 200)), (x, y + round(size * 0.02)), rounded_mask(t))
    sh = sh.filter(ImageFilter.GaussianBlur(size * 0.03))
    canvas.alpha_composite(sh)
    canvas.alpha_composite(tile(t), (x, y))
    return canvas


with_shadow(MASTER).save(OUT_PNG, optimize=True)

sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
frames = [tile(s) if s <= 32 else with_shadow(s) for s in sizes]
frames[-1].save(OUT_ICO, format="ICO", sizes=[(s, s) for s in sizes], append_images=frames[:-1])

chk = Image.open(OUT_ICO)
print("png", Image.open(OUT_PNG).size, "ico", sorted(chk.info["sizes"]))
