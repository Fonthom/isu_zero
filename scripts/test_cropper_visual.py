#!/usr/bin/env python3
"""
scripts/test_cropper_visual.py

Runs the full YOLO detection + crop pipeline on images in cropper/assets/
and saves annotated results to cropper/assets/output/ for visual inspection.

No Docker, no NATS, no database needed.

Usage:
    cd ~/workspace/isu_zero
    source cropper/.venv/bin/activate.fish
    python scripts/test_cropper_visual.py
"""

import sys
import os
from pathlib import Path

# Add cropper/ to path so we can import its modules
sys.path.insert(0, str(Path(__file__).parent.parent / "cropper"))

from PIL import Image, ImageDraw, ImageFont
from detector import detect_products, load_image, load_model
from hasher import compute_phash, crop_region, is_duplicate

ASSETS_DIR = Path(__file__).parent.parent / "cropper" / "assets"
OUTPUT_DIR = ASSETS_DIR / "output"

SUPPORTED = {".jpg", ".jpeg", ".png"}


def draw_detections(image: Image.Image, detections, output_path: Path):
    """Draw bounding boxes on the image and save it."""
    annotated = image.copy()
    draw = ImageDraw.Draw(annotated)

    for i, det in enumerate(detections):
        x, y, w, h = det.x, det.y, det.width, det.height
        draw.rectangle([x, y, x + w, y + h], outline="#00e5ff", width=3)
        label = f"#{i+1} {det.label} {det.confidence:.0%}"
        draw.rectangle([x, y - 20, x + len(label) * 7, y], fill="#00e5ff")
        draw.text((x + 2, y - 18), label, fill="#0a0e12")

    annotated.save(output_path)
    print(f"  saved annotated image → {output_path.name}")


def save_crops(image: Image.Image, detections, output_dir: Path, base_name: str):
    """Save each unique crop to disk."""
    existing_hashes = []
    saved = 0
    skipped = 0

    for i, det in enumerate(detections):
        crop = crop_region(image, det.x, det.y, det.width, det.height)
        phash = compute_phash(crop)

        if is_duplicate(phash, existing_hashes):
            print(f"  crop #{i+1} — duplicate, skipping")
            skipped += 1
            continue

        existing_hashes.append(phash)
        crop_path = output_dir / f"{base_name}_crop_{i+1:02d}.jpg"
        crop.save(crop_path, format="JPEG", quality=90)
        print(f"  crop #{i+1} — saved {crop_path.name} (hash: {phash})")
        saved += 1

    return saved, skipped


def main():
    images = [p for p in ASSETS_DIR.iterdir() if p.suffix.lower() in SUPPORTED]

    if not images:
        print(f"No images found in {ASSETS_DIR}")
        print("Add .jpg or .png shelf photos to cropper/assets/ first.")
        sys.exit(1)

    OUTPUT_DIR.mkdir(exist_ok=True)

    print("Loading YOLO model...")
    model = load_model()
    print(f"Model ready. Processing {len(images)} image(s)...\n")

    total_inserted = 0
    total_skipped  = 0

    for img_path in sorted(images):
        print(f"── {img_path.name}")
        image      = load_image(str(img_path))
        detections = detect_products(model, str(img_path))

        print(f"  {len(detections)} detection(s) found")

        if detections:
            from collections import Counter
            labels = Counter(d.label for d in detections)
            for label, count in labels.most_common():
                print(f"    {count}x {label}")

        if not detections:
            print("  no products detected — skipping\n")
            continue

        # Save annotated image with bounding boxes
        annotated_path = OUTPUT_DIR / f"{img_path.stem}_annotated{img_path.suffix}"
        draw_detections(image, detections, annotated_path)

        # Save unique crops
        inserted, skipped = save_crops(image, detections, OUTPUT_DIR, img_path.stem)
        total_inserted += inserted
        total_skipped  += skipped

        print(f"  {inserted} unique crops saved, {skipped} duplicates skipped\n")

    print(f"Done. {total_inserted} total crops saved to {OUTPUT_DIR}")
    print(f"Open the output/ folder to visually inspect results.")


if __name__ == "__main__":
    main()