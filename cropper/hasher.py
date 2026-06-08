import imagehash
from PIL import Image


# Hamming distance threshold — hashes with a distance below this
# are considered the same product. 10 is a good starting point:
# identical images score 0, slightly different lighting scores ~5,
# completely different products score 30+.
HASH_THRESHOLD = 10

def compute_phash(image: Image.Image) -> str:
    return str(imagehash.phash(image))


def hash_distance(hash_a: str, hash_b: str) -> int:
    return imagehash.hex_to_hash(hash_a) - imagehash.hex_to_hash(hash_b)


def is_duplicate(new_hash: str, existing_hashes: list[str]) -> bool:
    for existing in existing_hashes:
        if hash_distance(new_hash, existing) <= HASH_THRESHOLD:
            return True
    return False


def crop_region(image: Image.Image, x: int, y: int, width: int, height: int) -> Image.Image:
    return image.crop((x, y, x + width, y + height))