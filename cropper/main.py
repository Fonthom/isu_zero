import asyncio
import json
import logging
import os
from pathlib import Path

import nats

from detector import detect_products, load_image, load_model
from hasher import compute_phash, crop_region, is_duplicate
from storage import NewProduct, get_connection, get_existing_hashes, insert_photo, insert_product, get_waypoint

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)

STREAM_NAME  = "ISU_ZERO"
SUBJECT      = "photo.captured"
DURABLE_NAME = "cropper-photo-captured"
CROP_DIR     = os.environ.get("CROP_DIR", "/photos/crops")


async def handle_photo_captured(msg):
    await msg.ack()

    try:
        event = json.loads(msg.data.decode())
    except json.JSONDecodeError as e:
        logger.error(f"failed to decode event payload: {e}")
        return

    waypoint_id = event.get("waypoint_id")
    file_path   = event.get("file_path")

    if not waypoint_id or not file_path:
        logger.error(f"missing fields in event: {event}")
        return

    logger.info(f"processing photo from waypoint {waypoint_id}: {file_path}")

    conn = get_connection()
    try:
        waypoint = get_waypoint(conn, waypoint_id)
        if not waypoint:
            logger.error(f"waypoint not found: {waypoint_id}")
            return

        photo_id = insert_photo(conn, waypoint_id, file_path)
        logger.info(f"inserted photo record: {photo_id}")

        image   = load_image(file_path)
        detections = detect_products(model, file_path)

        if not detections:
            logger.info("no products detected in photo")
            return

        existing_hashes = get_existing_hashes(conn)
        logger.info(f"loaded {len(existing_hashes)} existing hashes for dedup")

        Path(CROP_DIR).mkdir(parents=True, exist_ok=True)

        inserted = 0
        skipped  = 0

        for detection in detections:
            crop = crop_region(image, detection.x, detection.y, detection.width, detection.height)
            phash = compute_phash(crop)

            if is_duplicate(phash, existing_hashes):
                logger.debug(f"duplicate detected, skipping: {phash}")
                skipped += 1
                continue

            crop_filename = f"{photo_id}_{detection.x}_{detection.y}.jpg"
            crop_path     = str(Path(CROP_DIR) / crop_filename)
            crop.save(crop_path, format="JPEG", quality=90)

            product_id = insert_product(conn, NewProduct(
                photo_id=photo_id,
                waypoint_id=waypoint_id,
                crop_x=detection.x,
                crop_y=detection.y,
                crop_width=detection.width,
                crop_height=detection.height,
                crop_path=crop_path,
                phash=phash,
            ))

            if product_id:
                existing_hashes.append(phash)
                inserted += 1
                logger.info(f"inserted product {product_id} from crop {crop_filename}")
            else:
                skipped += 1

        logger.info(f"photo processing complete — inserted: {inserted}, skipped: {skipped}")

    except Exception as e:
        logger.exception(f"error processing photo {file_path}: {e}")
    finally:
        conn.close()


async def main():
    nats_url = os.environ.get("NATS_URL", "nats://localhost:4222")
    logger.info(f"connecting to NATS at {nats_url}")

    nc = await nats.connect(nats_url)
    js = nc.jetstream()

    await js.subscribe(
        SUBJECT,
        durable=DURABLE_NAME,
        stream=STREAM_NAME,
        cb=handle_photo_captured,
    )

    logger.info(f"subscribed to {SUBJECT} as {DURABLE_NAME}")
    logger.info("cropper ready — waiting for photo events")

    try:
        await asyncio.Future()  # run forever
    except asyncio.CancelledError:
        pass
    finally:
        await nc.drain()
        logger.info("cropper shut down cleanly")


if __name__ == "__main__":
    logger.info("loading YOLO model...")
    model = load_model()
    logger.info("YOLO model ready")

    asyncio.run(main())