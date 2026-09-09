-- Migration: 000002_seed_categories.up.sql
-- Seed initial categories for VRChat assets

INSERT OR IGNORE INTO categories (name) VALUES
    ('Avatar'),
    ('Hair'),
    ('Clothes'),
    ('Shoes'),
    ('Accessory'),
    ('Gimmick'),
    ('Texture'),
    ('Material'),
    ('Shader'),
    ('Other');
