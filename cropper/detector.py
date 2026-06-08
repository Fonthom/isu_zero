import logging
from dataclasses import dataclass
from pathlib import Path

from PIL import Image
from ultralytics import YOLO

logger = logging.getLogger(__name__)

# YOLOv8 nano — fast and small, sufficient for shelf detection
MODEL_PATH = "yolov8n.pt"

# Minimum confidence score to accept a detection.
# Lower values catch more objects but increase false positives.
CONFIDENCE_THRESHOLD = 0.25

# Minimum pixel area for a detected region to be considered a product.
# Filters out tiny detections that are likely noise.
MIN_AREA = 1000


@dataclass
class Detection:
    x:          int
    y:          int
    width:      int
    height:     int
    confidence: float
    label:      str


def load_model() -> YOLO:
    logger.info(f"loading YOLO model from {MODEL_PATH}")
    return YOLO(MODEL_PATH)


def detect_products(model: YOLO, image_path: str) -> list[Detection]:
    path = Path(image_path)
    if not path.exists():
        logger.error(f"image not found: {image_path}")
        return []

    logger.info(f"running detection on {image_path}")
    results = model(image_path, conf=CONFIDENCE_THRESHOLD, verbose=False)

    detections = []
    for result in results:
        boxes = result.boxes
        names = result.names

        for box in boxes:
            # xyxy gives absolute pixel coordinates: x1, y1, x2, y2
            x1, y1, x2, y2 = box.xyxy[0].tolist()
            x1, y1, x2, y2 = int(x1), int(y1), int(x2), int(y2)

            width  = x2 - x1
            height = y2 - y1
            area   = width * height

            if area < MIN_AREA:
                logger.debug(f"skipping small detection: area={area}")
                continue

            confidence = float(box.conf[0])
            class_id   = int(box.cls[0])
            label      = names[class_id]

            detections.append(Detection(
                x=x1,
                y=y1,
                width=width,
                height=height,
                confidence=confidence,
                label=label,
            ))

    logger.info(f"found {len(detections)} detections in {image_path}")
    return detections


def load_image(image_path: str) -> Image.Image:
    try:
        return Image.open(image_path).convert("RGB")
    except Exception as e:
        raise ValueError(f"could not load image: {image_path}") from e