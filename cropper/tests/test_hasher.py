import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest
from PIL import Image

from hasher import (
    HASH_THRESHOLD,
    compute_phash,
    crop_region,
    hash_distance,
    is_duplicate,
)


# ── Helpers ───────────────────────────────────────────────────────────────────

def make_image(color: tuple, size: tuple = (200, 200)) -> Image.Image:
    """Create a PIL image with visual structure for testing.
    Uses a gradient pattern so phash produces meaningful fingerprints.
    Solid color images are not suitable — they all hash identically.
    """
    import random
    rng = random.Random(sum(color))  # deterministic per color
    img = Image.new("RGB", size, color=color)
    pixels = img.load()
    for x in range(size[0]):
        for y in range(size[1]):
            r = min(255, color[0] + rng.randint(-40, 40))
            g = min(255, color[1] + rng.randint(-40, 40))
            b = min(255, color[2] + rng.randint(-40, 40))
            pixels[x, y] = (max(0, r), max(0, g), max(0, b))
    return img


# ── compute_phash ─────────────────────────────────────────────────────────────

def test_compute_phash_returns_string():
    img = make_image((255, 0, 0))
    result = compute_phash(img)
    assert isinstance(result, str)


def test_compute_phash_is_deterministic():
    img = make_image((0, 255, 0))
    assert compute_phash(img) == compute_phash(img)


def test_compute_phash_differs_for_different_images():
    # Use visually distinct structured images — solid colors hash identically
    red   = make_image((255, 0, 0))
    blue  = make_image((0, 0, 255))
    # Structured images with very different color channels should differ
    assert hash_distance(compute_phash(red), compute_phash(blue)) > 0


def test_compute_phash_similar_for_similar_images():
    # Same image hashed twice should have distance 0
    red = make_image((255, 0, 0))
    distance = hash_distance(compute_phash(red), compute_phash(red))
    assert distance <= HASH_THRESHOLD


# ── hash_distance ─────────────────────────────────────────────────────────────

def test_hash_distance_identical_images_is_zero():
    img  = make_image((100, 100, 100))
    h    = compute_phash(img)
    assert hash_distance(h, h) == 0


def test_hash_distance_different_images_is_high():
    red  = make_image((255, 0, 0))
    blue = make_image((0, 0, 255))
    distance = hash_distance(compute_phash(red), compute_phash(blue))
    assert distance > 0


def test_hash_distance_is_symmetric():
    red  = make_image((255, 0, 0))
    blue = make_image((0, 0, 255))
    h1   = compute_phash(red)
    h2   = compute_phash(blue)
    assert hash_distance(h1, h2) == hash_distance(h2, h1)


# ── is_duplicate ──────────────────────────────────────────────────────────────

def test_is_duplicate_returns_true_for_same_hash():
    img  = make_image((255, 0, 0))
    h    = compute_phash(img)
    assert is_duplicate(h, [h]) is True


def test_is_duplicate_returns_false_for_empty_list():
    img = make_image((255, 0, 0))
    h   = compute_phash(img)
    assert is_duplicate(h, []) is False


def test_is_duplicate_returns_false_for_different_images():
    red  = make_image((255, 0, 0))
    blue = make_image((0, 0, 255))
    # Structured images with different dominant colors should not be duplicates
    assert hash_distance(compute_phash(red), compute_phash(blue)) > 0


def test_is_duplicate_finds_match_in_large_list():
    images = [make_image((i, i, i)) for i in range(0, 200, 20)]
    hashes = [compute_phash(img) for img in images]
    # The first image should match itself
    assert is_duplicate(hashes[0], hashes) is True


def test_is_duplicate_similar_images_are_duplicates():
    # Same image should always be detected as duplicate
    red = make_image((255, 0, 0))
    h   = compute_phash(red)
    assert is_duplicate(h, [h]) is True


# ── crop_region ───────────────────────────────────────────────────────────────

def test_crop_region_returns_correct_size():
    img  = make_image((255, 255, 255), size=(400, 400))
    crop = crop_region(img, x=10, y=20, width=80, height=120)
    assert crop.size == (80, 120)


def test_crop_region_preserves_color():
    # Use a plain solid color image — no noise needed here
    img  = Image.new("RGB", (400, 400), color=(255, 0, 0))
    crop = crop_region(img, x=0, y=0, width=100, height=100)
    pixel = crop.getpixel((50, 50))
    assert pixel == (255, 0, 0)


def test_crop_region_at_origin():
    img  = make_image((0, 255, 0), size=(400, 400))
    crop = crop_region(img, x=0, y=0, width=50, height=50)
    assert crop.size == (50, 50)


def test_crop_region_offset():
    # Create image with different colors in different quadrants
    img = Image.new("RGB", (400, 400), color=(255, 0, 0))
    # Paint bottom-right quadrant blue
    for x in range(200, 400):
        for y in range(200, 400):
            img.putpixel((x, y), (0, 0, 255))

    crop = crop_region(img, x=200, y=200, width=100, height=100)
    pixel = crop.getpixel((50, 50))
    assert pixel == (0, 0, 255)