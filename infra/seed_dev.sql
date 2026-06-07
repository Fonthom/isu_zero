-- Development seed data — loaded by docker-compose, never in tests

INSERT INTO waypoints (name, type, nav_x, nav_y) VALUES
    ('home_base',      'home',    0.0, 0.0),
    ('patrol_front',   'patrol',  1.5, 0.5),
    ('patrol_back',    'patrol',  1.5, 4.0),
    ('photo_aisle1a',  'photo',   2.0, 1.0),
    ('photo_aisle1b',  'photo',   2.0, 2.0),
    ('photo_aisle2a',  'photo',   2.0, 3.0),
    ('photo_aisle2b',  'photo',   2.0, 4.0)
ON CONFLICT DO NOTHING;