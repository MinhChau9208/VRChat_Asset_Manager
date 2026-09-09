-- Migration: 000003_add_favorite_to_assets.up.sql
-- Add is_favorite column and index to assets table for Milestone 7

ALTER TABLE assets ADD COLUMN is_favorite BOOLEAN NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_assets_is_favorite ON assets(is_favorite);
