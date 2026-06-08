import psycopg
import os
from dataclasses import dataclass
from typing import Optional


def get_connection():
    return psycopg.connect(os.environ["DATABASE_URL"])


@dataclass
class NewProduct:
    photo_id:    str
    waypoint_id: str
    crop_x:      int
    crop_y:      int
    crop_width:  int
    crop_height: int
    crop_path:   str
    phash:       str
    name:        Optional[str] = None


def insert_photo(conn, waypoint_id: str, file_path: str) -> str:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO photos (waypoint_id, file_path)
            VALUES (%s, %s)
            RETURNING id
            """,
            (waypoint_id, file_path),
        )
        photo_id = cur.fetchone()[0]
    conn.commit()
    return str(photo_id)


def get_existing_hashes(conn) -> list[str]:
    with conn.cursor() as cur:
        cur.execute("SELECT phash FROM products")
        return [row[0] for row in cur.fetchall()]


def insert_product(conn, product: NewProduct) -> Optional[str]:
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO products
                (photo_id, waypoint_id, name, crop_x, crop_y, crop_width, crop_height, crop_path, phash)
            VALUES
                (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (phash) DO NOTHING
            RETURNING id
            """,
            (
                product.photo_id,
                product.waypoint_id,
                product.name,
                product.crop_x,
                product.crop_y,
                product.crop_width,
                product.crop_height,
                product.crop_path,
                product.phash,
            ),
        )
        row = cur.fetchone()
    conn.commit()
    return str(row[0]) if row else None


def get_waypoint(conn, waypoint_id: str) -> Optional[dict]:
    with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
        cur.execute(
            "SELECT id, name, type, nav_x, nav_y FROM waypoints WHERE id = %s",
            (waypoint_id,),
        )
        row = cur.fetchone()
    return row if row else None