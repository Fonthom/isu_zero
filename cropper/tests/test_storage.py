import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

import pytest
import psycopg

from storage import (
    NewProduct,
    get_connection,
    get_existing_hashes,
    get_waypoint,
    insert_photo,
    insert_product,
)

# Known IDs from infra/seed_test.sql
WAYPOINT_ID_PHOTO  = "a0000000-0000-0000-0000-000000000003"  # photo_aisle1a
WAYPOINT_ID_HOME   = "a0000000-0000-0000-0000-000000000001"  # home_base
PHOTO_ID           = "b0000000-0000-0000-0000-000000000001"
UNKNOWN_ID         = "00000000-0000-0000-0000-000000000000"


# ── Helpers ───────────────────────────────────────────────────────────────────

@pytest.fixture
def conn():
    """
    Open a database connection for the test and close it after.
    Requires DATABASE_URL to be set in the environment.
    """
    url = os.environ.get("DATABASE_URL")
    if not url:
        pytest.fail("DATABASE_URL not set — run tests via scripts/test_cropper.sh")
    c = get_connection()
    yield c
    c.close()


# ── get_waypoint ──────────────────────────────────────────────────────────────

def test_get_waypoint_returns_known_waypoint(conn):
    waypoint = get_waypoint(conn, WAYPOINT_ID_PHOTO)
    assert waypoint is not None
    assert waypoint["name"] == "photo_aisle1a"
    assert waypoint["type"] == "photo"


def test_get_waypoint_returns_correct_coordinates(conn):
    waypoint = get_waypoint(conn, WAYPOINT_ID_PHOTO)
    assert waypoint["nav_x"] == 2.0
    assert waypoint["nav_y"] == 1.0


def test_get_waypoint_returns_none_for_unknown_id(conn):
    result = get_waypoint(conn, UNKNOWN_ID)
    assert result is None


def test_get_waypoint_home_base(conn):
    waypoint = get_waypoint(conn, WAYPOINT_ID_HOME)
    assert waypoint is not None
    assert waypoint["type"] == "home"


# ── insert_photo ──────────────────────────────────────────────────────────────

def test_insert_photo_returns_uuid(conn):
    photo_id = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/test_insert.jpg")
    assert isinstance(photo_id, str)
    assert len(photo_id) == 36  # UUID format: 8-4-4-4-12


def test_insert_photo_links_to_waypoint(conn):
    photo_id = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/test_link.jpg")
    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute("SELECT waypoint_id FROM photos WHERE id = %s", (photo_id,))
        row = cur.fetchone()
    assert row is not None
    assert str(row["waypoint_id"]) == WAYPOINT_ID_PHOTO


def test_insert_photo_records_file_path(conn):
    path = "/photos/test_filepath.jpg"
    photo_id = insert_photo(conn, WAYPOINT_ID_PHOTO, path)
    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute("SELECT file_path FROM photos WHERE id = %s", (photo_id,))
        row = cur.fetchone()
    assert row["file_path"] == path


# ── get_existing_hashes ───────────────────────────────────────────────────────

def test_get_existing_hashes_returns_list(conn):
    result = get_existing_hashes(conn)
    assert isinstance(result, list)


def test_get_existing_hashes_contains_seed_hashes(conn):
    hashes = get_existing_hashes(conn)
    # seed_test.sql inserts products with known hashes
    assert "aabbccdd11223344" in hashes


def test_get_existing_hashes_all_strings(conn):
    hashes = get_existing_hashes(conn)
    for h in hashes:
        assert isinstance(h, str)


# ── insert_product ────────────────────────────────────────────────────────────

def make_product(photo_id: str, phash: str, name: str = None) -> NewProduct:
    return NewProduct(
        photo_id=photo_id,
        waypoint_id=WAYPOINT_ID_PHOTO,
        crop_x=10,
        crop_y=20,
        crop_width=80,
        crop_height=120,
        crop_path=f"/photos/crops/test_{phash}.jpg",
        phash=phash,
        name=name,
    )


def test_insert_product_returns_uuid(conn):
    photo_id   = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p1.jpg")
    product_id = insert_product(conn, make_product(photo_id, "unique_hash_001"))
    assert isinstance(product_id, str)
    assert len(product_id) == 36


def test_insert_product_with_name(conn):
    photo_id   = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p2.jpg")
    product_id = insert_product(conn, make_product(photo_id, "unique_hash_002", name="Test Cola"))
    assert product_id is not None

    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute("SELECT name FROM products WHERE id = %s", (product_id,))
        row = cur.fetchone()
    assert row["name"] == "Test Cola"


def test_insert_product_without_name_is_null(conn):
    photo_id   = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p3.jpg")
    product_id = insert_product(conn, make_product(photo_id, "unique_hash_003"))
    assert product_id is not None

    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute("SELECT name FROM products WHERE id = %s", (product_id,))
        row = cur.fetchone()
    assert row["name"] is None


def test_insert_product_duplicate_phash_returns_none(conn):
    photo_id = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p4.jpg")
    product  = make_product(photo_id, "unique_hash_004")

    first  = insert_product(conn, product)
    second = insert_product(conn, product)

    assert first  is not None
    assert second is None  # ON CONFLICT DO NOTHING


def test_insert_product_stores_crop_coordinates(conn):
    photo_id   = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p5.jpg")
    product_id = insert_product(conn, make_product(photo_id, "unique_hash_005"))

    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute(
            "SELECT crop_x, crop_y, crop_width, crop_height FROM products WHERE id = %s",
            (product_id,),
        )
        row = cur.fetchone()
    assert row["crop_x"]      == 10
    assert row["crop_y"]      == 20
    assert row["crop_width"]  == 80
    assert row["crop_height"] == 120


def test_insert_product_links_to_waypoint(conn):
    photo_id   = insert_photo(conn, WAYPOINT_ID_PHOTO, "/photos/p6.jpg")
    product_id = insert_product(conn, make_product(photo_id, "unique_hash_006"))

    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute("SELECT waypoint_id FROM products WHERE id = %s", (product_id,))
        row = cur.fetchone()
    assert str(row["waypoint_id"]) == WAYPOINT_ID_PHOTO