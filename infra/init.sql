CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS waypoints (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    type       TEXT NOT NULL CHECK (type IN ('home', 'patrol', 'photo')),
    nav_x      FLOAT NOT NULL,
    nav_y      FLOAT NOT NULL
);

CREATE TABLE IF NOT EXISTS photos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    waypoint_id UUID NOT NULL REFERENCES waypoints(id),
    file_path   TEXT NOT NULL,
    taken_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id     UUID NOT NULL REFERENCES photos(id),
    waypoint_id  UUID NOT NULL REFERENCES waypoints(id),
    name         TEXT,
    crop_x       INT NOT NULL,
    crop_y       INT NOT NULL,
    crop_width   INT NOT NULL,
    crop_height  INT NOT NULL,
    crop_path    TEXT NOT NULL,
    phash        TEXT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS interactions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id       UUID REFERENCES products(id),
    query_text       TEXT NOT NULL,
    outcome          TEXT NOT NULL CHECK (outcome IN ('navigated', 'not_found')),
    duration_seconds INT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_phash ON products(phash);
CREATE INDEX IF NOT EXISTS idx_products_name ON products USING gin(to_tsvector('english', coalesce(name, '')));
CREATE INDEX IF NOT EXISTS idx_interactions_created_at ON interactions(created_at);
CREATE INDEX IF NOT EXISTS idx_photos_waypoint ON photos(waypoint_id);

-- Seed waypoints for demo store layout
INSERT INTO waypoints (name, type, nav_x, nav_y) VALUES
    ('home_base',      'home',    0.0, 0.0),
    ('patrol_front',   'patrol',  1.5, 0.5),
    ('patrol_back',    'patrol',  1.5, 4.0),
    ('photo_aisle1a',  'photo',   2.0, 1.0),
    ('photo_aisle1b',  'photo',   2.0, 2.0),
    ('photo_aisle2a',  'photo',   2.0, 3.0),
    ('photo_aisle2b',  'photo',   2.0, 4.0)
ON CONFLICT DO NOTHING;