-- Migration: 000005_scanner.up.sql
-- Milestone 8: filesystem scanner settings, ignored paths and per-draft scan hints.

-- JSON describing why the scanner suggested what it did (sources, candidates).
ALTER TABLE assets ADD COLUMN scan_info TEXT NOT NULL DEFAULT '';

-- Small key/value store for app settings (e.g. "scanner" -> JSON config).
CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Paths the user chose to ignore in the review screen; re-scans skip them.
CREATE TABLE IF NOT EXISTS scan_ignored (
    path TEXT PRIMARY KEY COLLATE NOCASE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
