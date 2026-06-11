import logging
from dataclasses import dataclass
from pathlib import Path

from PIL import Image
from ultralytics import YOLO

logger = logging.getLogger(__name__)

# YOLOv8 small — better detection than nano for cluttered shelf scenes
MODEL_PATH = "cropper/best.pt"

# Minimum confidence score to accept a detection.
# Lower values catch more objects but increase false positives.
# 0.15 is more permissive — needed for grocery packaging not in COCO classes.
CONFIDENCE_THRESHOLD = 0.125

# Minimum pixel area for a detected region to be considered a product.
# Lowered to 500 to catch smaller items like cans in dense shelf arrangements.
MIN_AREA = 500


@dataclass
class Detection:
    x:          int
    y:          int
    width:      int
    height:     int
    confidence: float
    label:      str


def load_model() -> YOLO:
    """
    Load the YOLOv8 model. Called once at startup.
    The weights are baked into the image at build time.
    """
    logger.info(f"loading YOLO model from {MODEL_PATH}")
    return YOLO(MODEL_PATH)


def detect_products(model: YOLO, image_path: str) -> list[Detection]:
    """
    Run YOLO inference on a shelf photo and return a list of detections.
    Each detection represents one product instance found in the image.
    Duplicate instances of the same product (e.g. six cans in a row)
    are all returned here — deduplication happens in the cropper pipeline
    using perceptual hashing.
    """
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
    """
    Load an image from disk as a PIL Image.
    """
    try:
        return Image.open(image_path).convert("RGB")
    except Exception as e:
        raise ValueError(f"could not load image: {image_path}") from e
