# REST API

The Go backend serves JSON on `http://localhost:8080` (CORS allows `http://localhost:3000`).
Errors are returned as `{"error": "message"}` with a matching HTTP status.

## Health

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Server health check → `{"status":"ok"}` |
| `GET` | `/api/health/db` | SQLite connectivity |

## Assets

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/assets` | List assets. Query: `search`, `category` (id or name, includes subcategories), `tags` (comma separated, all must match), `favorite`, `has_preview`, `has_booth`, `local_status` (`available` / `missing` / `not_specified`), `sort` (`recent` / `updated` / `name_asc` / `name_desc`), `status` (`active` default / `draft` / `all`), `compatible_with` (avatar asset id) |
| `GET` | `/api/assets/:id` | One asset with category, tags, `files`, `compatible_avatars` and `scan_info` |
| `POST` | `/api/assets` | Create an asset |
| `PUT` | `/api/assets/:id` | Update an asset. Omitting `preview_path`, `tags`, `compatible_avatars` or `status` keeps the current value |
| `DELETE` | `/api/assets/:id` | Delete the library entry and its preview copy (files on disk are never touched) |
| `POST` | `/api/assets/bulk` | Same change for many assets: `set_category` + `category_id`, `add_tags`, `add_compatible_avatars`, `is_favorite`. Tags and avatars are only added |
| `POST` | `/api/assets/:id/favorite` | Set `{"is_favorite": true}` or toggle with `{}` |
| `GET` | `/api/stats` | Unfiltered counts: `total`, `favorites`, `drafts`, `by_category` |

### Files on disk

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/assets/:id/files` | Link a folder / archive / package: `{"path", "version"?, "kind"?}` (kind is inferred) |
| `DELETE` | `/api/assets/:id/files/:fileId` | Unlink a file; the next file becomes primary if needed |
| `GET` | `/api/assets/:id/status` | `{"exists": bool}` for the primary `local_path` |
| `POST` | `/api/assets/batch-status` | `{"ids": [...]}` → `{"statuses": {"1": true}}` |
| `POST` | `/api/assets/:id/open-folder` | Open the location in the OS file explorer |
| `POST` | `/api/filesystem/pick-folder` | Native folder picker (Windows only; returns `{"path": ""}` elsewhere) |

### Preview images

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/assets/:id/preview` | `multipart/form-data` upload, JPEG / PNG / WebP, max 10MB, type checked by content |
| `GET` | `/api/assets/:id/preview` | Serve the image |
| `DELETE` | `/api/assets/:id/preview` | Remove the preview |

## Categories & tags

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/categories` | Categories in display order, each with `parent_id` and `sort_order` |
| `POST` | `/api/categories` | `{"name", "parent_id"?, "sort_order"?}` (two levels max, unique names) |
| `PUT` | `/api/categories/:id` | Rename / re-parent / reorder |
| `DELETE` | `/api/categories/:id` | Delete a category without subcategories; its assets become uncategorized |
| `GET` | `/api/tags` | All tags |
| `POST` | `/api/tags` | Create or get a tag |

## Scanner

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/scanner/config` | `roots`, `ignore`, `archive_dirs`, `folder_map`, `known_dependencies` |
| `PUT` | `/api/scanner/config` | Save the configuration |
| `POST` | `/api/scanner/scan` | Scan the roots → `{"groups", "created", "attached", "already_linked", "ignored", "warnings", "duration_ms"}` |
| `POST` | `/api/scanner/accept` | `{"asset_ids": [...]}` turns drafts into library assets |
| `POST` | `/api/scanner/ignore` | Delete drafts and skip their paths in later scans |
| `GET` | `/api/scanner/ignored` | Ignored paths |
| `DELETE` | `/api/scanner/ignored?path=` | Scan an ignored path again |

## BOOTH import

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/booth/lookup?url=` | Item URL or id → suggestions: `name`, `author`, `booth_url`, `category_id`, `booth_category`, `tags`, `compatible_avatars`, `images`. Cached 7 days; `refresh=1` re-fetches |
| `POST` | `/api/booth/preview` | `{"asset_id", "url"}` downloads a BOOTH image (pximg.net only) as the preview |
| `POST` | `/api/booth/apply` | `{"asset_ids", "include_tags"?}` fills assets from their BOOTH links: replaces the name, fills empty author / category / preview, merges compatible avatars |

## Example: create an asset

```http
POST /api/assets
Content-Type: application/json

{
  "name": "Manuka Avatar",
  "category_id": 1,
  "author": "Jingo Channel",
  "booth_url": "https://booth.pm/en/items/4394473",
  "local_path": "D:/VRChat/Avatars/Manuka",
  "tags": ["avatar", "physbone"],
  "compatible_avatars": []
}
```

Response `201 Created`:

```json
{
  "id": 1,
  "name": "Manuka Avatar",
  "category_id": 1,
  "category": { "id": 1, "name": "Avatar" },
  "author": "Jingo Channel",
  "booth_url": "https://booth.pm/en/items/4394473",
  "local_path": "D:/VRChat/Avatars/Manuka",
  "preview_path": "",
  "description": "",
  "tags": ["avatar", "physbone"],
  "is_favorite": false,
  "status": "active",
  "local_file_exists": false,
  "files": [{ "id": 1, "path": "D:/VRChat/Avatars/Manuka", "kind": "folder", "version": "", "exists": false }],
  "created_at": "2026-09-09T06:20:00Z",
  "updated_at": "2026-09-09T06:20:00Z"
}
```
