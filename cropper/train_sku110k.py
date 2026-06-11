#!/usr/bin/env python3
"""
scripts/train_sku110k.py

Fine-tunes YOLOv8s on the SKU-110K retail shelf dataset.
Requires an NVIDIA GPU with CUDA. An RTX 5060 should complete in 2-4 hours.

Usage:
    cd ~/workspace/isu_zero
    source cropper/.venv/bin/activate.fish
    python scripts/train_sku110k.py

Output:
    runs/detect/sku110k/weights/best.pt  ← use this as your model
"""

from ultralytics import YOLO
import torch

# Training configuration
MODEL       = "yolov8s.pt"   # start from pretrained COCO weights
DATA        = "SKU-110K.yaml" # Ultralytics built-in dataset config
EPOCHS      = 100
IMAGE_SIZE  = 640
BATCH_SIZE  = 16              # safe starting point — YOLO will auto-adjust if you set to -1
PROJECT     = "runs/detect"
NAME        = "sku110k"
WORKERS     = 2


def main():
    # Determine device
    if torch.cuda.is_available():
        device = 0
        print(f"✅ GPU detected: {torch.cuda.get_device_name(0)}")
        print(f"   VRAM: {torch.cuda.get_device_properties(0).total_memory / 1e9:.1f} GB")
    else:
        device = "cpu"
        print("⚠️  No GPU detected! Training will be very slow on CPU.")
        print("   Make sure CUDA is installed and PyTorch has GPU support.")

    print("Loading base model...")
    model = YOLO(MODEL)

    print(f"Starting training on SKU-110K for {EPOCHS} epochs...")
    print(f"Batch size: {BATCH_SIZE}, Image size: {IMAGE_SIZE}, Device: {device}")
    print(f"Output will be saved to {PROJECT}/{NAME}/\n")

    results = model.train(
        data=DATA,
        epochs=EPOCHS,
        imgsz=IMAGE_SIZE,
        batch=BATCH_SIZE,
        device=device,           # ← Explicit device
        project=PROJECT,
        name=NAME,
        workers=WORKERS,
        # Performance & stability improvements
        cache=False,              # SKU-110K is large, so we won't cache in RAM
        patience=50,             # Early stopping
        # Augmentation — tuned for retail shelves
        hsv_h=0.015,
        hsv_s=0.7,
        hsv_v=0.4,
        flipud=0.0,              # shelves are never upside down
        fliplr=0.5,
        mosaic=1.0,
        mixup=0.1,               # mild mixup helps with dense scenes
        copy_paste=0.1,          # helps with occlusion
        # Training stability
        optimizer="auto",
        save=True,
        save_period=-1,
        plots=True,
    )

    print("\n✅ Training complete.")
    print(f"Best weights saved to: {PROJECT}/{NAME}/weights/best.pt")
    print("\nNext step: copy best.pt to cropper/ and update MODEL_PATH in detector.py")


if __name__ == "__main__":
    main()