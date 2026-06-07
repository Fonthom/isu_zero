-- Test seed data for ISU-Zero
-- Used exclusively by scripts/test.sh
-- Never loaded in production

-- ── Waypoints ─────────────────────────────────────────────────────────────────
INSERT INTO waypoints (id, name, type, nav_x, nav_y) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'home_base',     'home',    0.0, 0.0),
    ('a0000000-0000-0000-0000-000000000002', 'patrol_front',  'patrol',  1.5, 0.5),
    ('a0000000-0000-0000-0000-000000000003', 'photo_aisle1a', 'photo',   2.0, 1.0),
    ('a0000000-0000-0000-0000-000000000004', 'photo_aisle1b', 'photo',   2.0, 2.0),
    ('a0000000-0000-0000-0000-000000000005', 'photo_aisle2a', 'photo',   2.0, 3.0)
ON CONFLICT DO NOTHING;

-- ── Photos ────────────────────────────────────────────────────────────────────
INSERT INTO photos (id, waypoint_id, file_path, taken_at) VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000003',
        '/photos/aisle1a.jpg',
        NOW()
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000004',
        '/photos/aisle1b.jpg',
        NOW()
    ),
    (
        'b0000000-0000-0000-0000-000000000003',
        'a0000000-0000-0000-0000-000000000005',
        '/photos/aisle2a.jpg',
        NOW()
    )
ON CONFLICT DO NOTHING;

-- ── Products ──────────────────────────────────────────────────────────────────
-- Named products (searchable)
INSERT INTO products (id, photo_id, waypoint_id, name, crop_x, crop_y, crop_width, crop_height, crop_path, phash) VALUES
    (
        'c0000000-0000-0000-0000-000000000001',
        'b0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000003',
        'Coca-Cola 500ml',
        10, 20, 80, 120,
        '/photos/crops/coca_cola_500ml.jpg',
        'aabbccdd11223344'
    ),
    (
        'c0000000-0000-0000-0000-000000000002',
        'b0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000003',
        'Orange Juice 1L',
        100, 20, 80, 120,
        '/photos/crops/orange_juice_1l.jpg',
        'bbccddee22334455'
    ),
    (
        'c0000000-0000-0000-0000-000000000003',
        'b0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000004',
        'Lay''s Classic',
        10, 20, 80, 120,
        '/photos/crops/lays_classic.jpg',
        'ccddeeff33445566'
    ),
    (
        'c0000000-0000-0000-0000-000000000004',
        'b0000000-0000-0000-0000-000000000003',
        'a0000000-0000-0000-0000-000000000005',
        'Whole Milk 1L',
        10, 20, 80, 120,
        '/photos/crops/whole_milk_1l.jpg',
        'ddeeff0044556677'
    )
ON CONFLICT DO NOTHING;

-- Unnamed product (not yet tagged — should not appear in search results)
INSERT INTO products (id, photo_id, waypoint_id, name, crop_x, crop_y, crop_width, crop_height, crop_path, phash) VALUES
    (
        'c0000000-0000-0000-0000-000000000005',
        'b0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000004',
        NULL,
        200, 20, 80, 120,
        '/photos/crops/unknown_001.jpg',
        'eeff001155667788'
    )
ON CONFLICT DO NOTHING;

-- ── Interactions ──────────────────────────────────────────────────────────────
INSERT INTO interactions (id, product_id, query_text, outcome, duration_seconds, created_at) VALUES
    (
        'd0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000001',
        'cola',
        'navigated',
        45,
        NOW() - INTERVAL '2 hours'
    ),
    (
        'd0000000-0000-0000-0000-000000000002',
        NULL,
        'batteries',
        'not_found',
        0,
        NOW() - INTERVAL '1 hour'
    ),
    (
        'd0000000-0000-0000-0000-000000000003',
        'c0000000-0000-0000-0000-000000000003',
        'chips',
        'navigated',
        38,
        NOW() - INTERVAL '30 minutes'
    )
ON CONFLICT DO NOTHING;