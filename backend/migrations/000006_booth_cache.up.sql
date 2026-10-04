-- Migration: 000006_booth_cache.up.sql
-- Milestone 9: cache of BOOTH item metadata so items are fetched at most once a week.

CREATE TABLE IF NOT EXISTS booth_cache (
    item_id TEXT PRIMARY KEY,
    data TEXT NOT NULL,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
