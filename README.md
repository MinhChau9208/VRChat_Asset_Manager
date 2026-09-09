# VRChat Asset Manager

A local-first personal asset catalog and management tool for VRChat creators and users. Organizes avatars, clothing, accessories, hair, shaders, gimmicks, and materials while preserving links to BOOTH and local files.

## Technology Stack

- **Frontend**: Next.js 16 (App Router), TypeScript, Tailwind CSS
- **Backend**: Go 1.26 (Standard Library HTTP server, REST API)
- **Database**: SQLite (via pure-Go driver `modernc.org/sqlite`)
- **Storage**: Local filesystem for asset files and preview images

## Repository Structure

```text
vrchat-asset-manager/
├── frontend/          # Next.js frontend application (TypeScript, Tailwind CSS)
│   ├── app/           # App router pages, layouts, and styles
│   ├── .env.example   # Example frontend environment file
│   └── .env.local     # Local environment configuration (gitignored)
├── backend/           # Go REST API server
│   ├── cmd/           # CLI utilities (e.g. cmd/verify verification tool)
│   ├── internal/      # Internal packages (asset CRUD, database connection)
│   ├── migrations/    # Explicit SQL migrations (*.up.sql)
│   ├── go.mod
│   └── main.go
├── data/              # Local runtime data (app.db, previews/ - gitignored)
├── docs/              # Project documentation and architecture records
├── PROJECT_SPEC.md    # Source of truth project specification and milestone roadmap
├── README.md          # Setup and developer documentation
└── .gitignore         # Repository ignore rules
```

## Getting Started

### Prerequisites

- [Node.js](https://nodejs.org/) (v18+ recommended)
- [Go](https://go.dev/) (v1.22+ recommended)

### 1. Running the Backend

Open a terminal and run:

```bash
cd backend
go run .
```

On startup, the backend automatically:
1. Connects to SQLite at `data/app.db` (or `../data/app.db`).
2. Applies all pending `.up.sql` migrations from `backend/migrations/`.
3. Seeds default VRChat asset categories (`Avatar`, `Hair`, `Clothes`, etc.).
4. Starts the HTTP server on port `8080`.

- **Backend Base URL**: [http://localhost:8080](http://localhost:8080)
- **Server Health**: [http://localhost:8080/health](http://localhost:8080/health)
- **Database Health**: [http://localhost:8080/api/health/db](http://localhost:8080/api/health/db)

### 2. Running the Frontend

In a separate terminal, install dependencies and start the Next.js development server:

```bash
cd frontend
npm install
npm run dev
```

The frontend application will start on:
- **URL**: [http://localhost:3000](http://localhost:3000)

### Environment Configuration

The frontend communicates with the Go backend using the `NEXT_PUBLIC_API_URL` environment variable:

- Default: `http://localhost:8080`
- To override, set `NEXT_PUBLIC_API_URL` in `frontend/.env.local`:
  ```env
  NEXT_PUBLIC_API_URL=http://localhost:8080
  ```

## Development URLs & API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Server health check |
| `GET` | `/api/health/db` | SQLite database connectivity health |
| `GET` | `/api/categories` | List all asset categories |
| `GET` | `/api/assets` | List assets (supports `search`, `category`, `tags`, `favorite`, `has_preview`, `has_booth`, `local_status`, `sort`) |
| `GET` | `/api/assets/:id` | Retrieve single asset with category and tags |
| `GET` | `/api/assets/:id/status` | Check if asset's `local_path` exists on disk |
| `POST` | `/api/assets/batch-status` | Batch check `local_path` existence on disk for multiple asset IDs |
| `POST` | `/api/assets/:id/favorite` | Toggle or set asset favorite status (`is_favorite`) |
| `POST` | `/api/assets/:id/open-folder` | Open asset's local folder in OS file explorer |
| `POST` | `/api/assets/:id/preview` | Upload a preview image (JPEG, PNG, WebP) |
| `GET` | `/api/assets/:id/preview` | Serve the asset preview image |
| `DELETE` | `/api/assets/:id/preview` | Delete the asset preview image |
| `POST` | `/api/assets` | Create a new asset |
| `PUT` | `/api/assets/:id` | Update an existing asset |
| `DELETE` | `/api/assets/:id` | Delete an asset record (never touches `local_path`) |
| `GET` | `/api/tags` | List all existing tags |
| `POST` | `/api/tags` | Create or get tag |
| `POST` | `/api/filesystem/pick-folder` | Open native Windows folder browser (local only) |

### API Request & Response Examples

#### Toggle Favorite (`POST /api/assets/:id/favorite`)
Optional JSON body: `{"is_favorite": true}` or empty `{}` to toggle.
Response (`200 OK`):
```json
{
  "id": 1,
  "name": "Manuka Avatar",
  "is_favorite": true,
  "updated_at": "2026-09-09T08:30:00Z"
}
```

#### Batch Check Filesystem Status (`POST /api/assets/batch-status`)
Request body:
```json
{
  "ids": [1, 2, 3]
}
```
Response (`200 OK`):
```json
{
  "statuses": {
    "1": true,
    "2": false,
    "3": false
  }
}
```

#### Upload Preview (`POST /api/assets/:id/preview`)
Accepts `multipart/form-data` with one image file (`image/jpeg`, `image/png`, `image/webp`, max 10MB).
Response (`200 OK`):
```json
{
  "id": 1,
  "name": "Manuka Avatar",
  "preview_path": "data/previews/1.webp",
  "updated_at": "2026-09-09T07:20:00Z"
}
```

#### Retrieve Preview (`GET /api/assets/:id/preview`)
Returns image binary stream with appropriate `Content-Type` (`image/jpeg`, `image/png`, or `image/webp`) and caching headers.

#### Delete Preview (`DELETE /api/assets/:id/preview`)
Response (`200 OK`):
```json
{
  "message": "preview deleted"
}
```

#### Check File Existence Status (`GET /api/assets/:id/status`)
Response (`200 OK`):
```json
{
  "exists": true
}
```

#### Open Asset Folder (`POST /api/assets/:id/open-folder`)
Response (`200 OK`):
```json
{
  "status": "ok"
}
```

#### Create Asset (`POST /api/assets`)
```json
{
  "name": "Manuka Avatar",
  "category_id": 1,
  "author": "Jingo Channel",
  "booth_url": "https://booth.pm/en/items/4394473",
  "local_path": "D:/VRChat/Avatars/Manuka",
  "description": "Base model for Manuka",
  "tags": ["avatar", "female", "physbone"]
}
```

Response (`201 Created`):
```json
{
  "id": 1,
  "name": "Manuka Avatar",
  "category_id": 1,
  "category": {
    "id": 1,
    "name": "Avatar"
  },
  "author": "Jingo Channel",
  "booth_url": "https://booth.pm/en/items/4394473",
  "local_path": "D:/VRChat/Avatars/Manuka",
  "preview_path": "",
  "description": "Base model for Manuka",
  "tags": ["avatar", "female", "physbone"],
  "is_favorite": false,
  "created_at": "2026-09-09T06:20:00Z",
  "updated_at": "2026-09-09T06:20:00Z"
}
```

#### Update Asset (`PUT /api/assets/:id`)
```json
{
  "name": "Manuka Avatar v2",
  "category_id": 1,
  "author": "Jingo Channel",
  "tags": ["avatar", "female", "updated"]
}
```

#### Delete Asset (`DELETE /api/assets/:id`)
Response (`200 OK`):
```json
{
  "message": "asset deleted"
}
```

## Running Tests & Verification

### Run Automated Backend Tests
```bash
cd backend
go test -v ./...
```

### Run End-to-End Verification Tool
With the backend running:
```bash
cd backend
go run ./cmd/verify
```

### Run Milestone 7 Integration Tests
```bash
node scratch/verify_m7.mjs
```

### Seed Development Sample Assets
To populate sample VRChat assets for visual testing:
```bash
cd backend
go run ./cmd/seed
```

## Milestone Status

- [x] **Milestone 0**: Project setup (Next.js, Go HTTP server, health check, CORS, developer docs)
- [x] **Milestone 1**: Database (SQLite setup, migrations, initial tables, seed categories, DB health check)
- [x] **Milestone 2**: Asset CRUD (REST API, validation, tag management, filter queries, automated test suite)
- [x] **Milestone 3**: Asset Grid UI (Next.js asset library, responsive card grid, dynamic categories, search, empty/loading states)
- [x] **Milestone 4**: Asset Details (Read-only detail page, preview/metadata inspection, local file existence check, OS folder launcher)
- [x] **Milestone 5**: Preview Upload (Local preview storage, MIME signature validation, image serving, replacement & deletion, AssetCard & AssetDetail UI)
- [x] **Milestone 6**: Asset Management UI (Add/edit/delete assets, tag auto-completion & creation, category assignment, native folder picker, safe deletion)
- [x] **Milestone 7**: Library Quality of Life (Search across name/author/tags, multi-tag filter, sorting, favorites persistence & filtering, local file status filters, URL query persistence)
- [ ] **Milestone 8**: Filesystem Scanner / Later Milestones (Deferred)
