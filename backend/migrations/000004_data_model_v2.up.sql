-- Migration: 000004_data_model_v2.up.sql
-- Milestone 7: category tree, asset files & versions, avatar compatibility, draft status.
-- Existing assets keep their category; only Material (merged into Texture & Material) changes id.

-- 1. Category tree --------------------------------------------------------

ALTER TABLE categories ADD COLUMN parent_id INTEGER REFERENCES categories(id) ON DELETE RESTRICT;
ALTER TABLE categories ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);

-- Renames / merges of the original seed categories
UPDATE categories SET name = 'Texture & Material' WHERE name = 'Texture';
UPDATE categories SET name = 'Tool & Shader' WHERE name = 'Shader';

INSERT OR IGNORE INTO categories (name) VALUES ('Texture & Material');
UPDATE assets
SET category_id = (SELECT id FROM categories WHERE name = 'Texture & Material')
WHERE category_id = (SELECT id FROM categories WHERE name = 'Material');
DELETE FROM categories WHERE name = 'Material';

-- New top-level categories
INSERT OR IGNORE INTO categories (name) VALUES
    ('Outfit'),
    ('Face'),
    ('Animation'),
    ('World'),
    ('Audio');

-- New child categories
INSERT OR IGNORE INTO categories (name) VALUES
    ('Ears & Tail'),
    ('Eyes'),
    ('Expression'),
    ('Makeup'),
    ('Prop');

-- Top-level order
UPDATE categories SET sort_order = 10  WHERE name = 'Avatar';
UPDATE categories SET sort_order = 20  WHERE name = 'Outfit';
UPDATE categories SET sort_order = 30  WHERE name = 'Hair';
UPDATE categories SET sort_order = 40  WHERE name = 'Accessory';
UPDATE categories SET sort_order = 50  WHERE name = 'Face';
UPDATE categories SET sort_order = 60  WHERE name = 'Gimmick';
UPDATE categories SET sort_order = 70  WHERE name = 'Animation';
UPDATE categories SET sort_order = 80  WHERE name = 'Texture & Material';
UPDATE categories SET sort_order = 90  WHERE name = 'Tool & Shader';
UPDATE categories SET sort_order = 100 WHERE name = 'World';
UPDATE categories SET sort_order = 110 WHERE name = 'Audio';
UPDATE categories SET sort_order = 999 WHERE name = 'Other';

-- Children
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Outfit'),    sort_order = 10 WHERE name = 'Clothes';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Outfit'),    sort_order = 20 WHERE name = 'Shoes';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Accessory'), sort_order = 10 WHERE name = 'Ears & Tail';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Face'),      sort_order = 10 WHERE name = 'Eyes';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Face'),      sort_order = 20 WHERE name = 'Expression';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Face'),      sort_order = 30 WHERE name = 'Makeup';
UPDATE categories SET parent_id = (SELECT id FROM categories WHERE name = 'Gimmick'),   sort_order = 10 WHERE name = 'Prop';

-- 2. Draft status ---------------------------------------------------------

ALTER TABLE assets ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'draft'));

CREATE INDEX IF NOT EXISTS idx_assets_status ON assets(status);

-- 3. Asset files & versions -----------------------------------------------

CREATE TABLE IF NOT EXISTS asset_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id INTEGER NOT NULL,
    path TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL DEFAULT 'folder' CHECK (kind IN ('folder', 'archive', 'unitypackage', 'file')),
    version TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_asset_files_asset_id ON asset_files(asset_id);

-- Every existing local_path becomes the asset's first file.
INSERT OR IGNORE INTO asset_files (asset_id, path, kind)
SELECT id, local_path,
    CASE
        WHEN lower(local_path) LIKE '%.zip' OR lower(local_path) LIKE '%.7z' OR lower(local_path) LIKE '%.rar' THEN 'archive'
        WHEN lower(local_path) LIKE '%.unitypackage' THEN 'unitypackage'
        ELSE 'folder'
    END
FROM assets
WHERE local_path IS NOT NULL AND trim(local_path) != '';

-- 4. Avatar compatibility -------------------------------------------------

-- avatar_name is always filled (copied from the avatar asset when linked), so the
-- relation survives if the avatar asset is deleted.
CREATE TABLE IF NOT EXISTS asset_compat (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_id INTEGER NOT NULL,
    avatar_asset_id INTEGER,
    avatar_name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE,
    FOREIGN KEY (avatar_asset_id) REFERENCES assets(id) ON DELETE SET NULL,
    UNIQUE (asset_id, avatar_name COLLATE NOCASE)
);

CREATE INDEX IF NOT EXISTS idx_asset_compat_avatar_asset_id ON asset_compat(avatar_asset_id);
