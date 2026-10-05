<div align="center">

# VRChat Asset Manager

**A local-first library for the VRChat assets you bought on BOOTH.**
See what you own, what it looks like, where it lives on disk, and which avatar it fits.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-16-000000?logo=nextdotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-local-003B57?logo=sqlite&logoColor=white)
![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-4-06B6D4?logo=tailwindcss&logoColor=white)

<picture>
  <source media="(prefers-color-scheme: light)" srcset="docs/screenshots/library-light.png">
  <img alt="Asset library with square preview cards, category sidebar and avatar shortcuts" src="docs/screenshots/library-dark.png" width="900">
</picture>

</div>

---

After a while, a VRChat folder turns into hundreds of zips and half-remembered folder names:
`hamanosis_Small_Lady_Kipfel`, `BukiyouTwinTail_1.1.0.zip`, `LEGACY/Gothic_Doll.zip`…
Which outfit was for which avatar? Where is the BOOTH page? Did I already extract that one?

VRChat Asset Manager answers those questions without moving a single file. It scans the folders you
already have, groups extracted folders with their original zips and older versions, pulls names and
preview images from BOOTH when you ask, and shows everything as a visual library on `localhost`.

## Features

### 📚 A visual library
Square preview cards (BOOTH images are 1:1) in three sizes or a compact list, a two-level category
tree, tags, favorites, search by name / author / tag, and filters such as *missing from disk* or
*has BOOTH link*. Click a card to open it in a side drawer without losing your place.

<p align="center"><img alt="Asset details in the side drawer" src="docs/screenshots/drawer.png" width="800"></p>

### 🔍 Scan your existing folders
Point the scanner at your asset folder and it proposes one draft per asset:

- **Level 1 folders are categories** (`Clothes`, `Hair`, `Models`…), level 2 entries are assets.
- **Zips are matched to their extracted folder** even when the names differ in case, width,
  separators or version (`Gothic Doll` ↔ `LEGACY/Gothic_Doll.zip`), and even across categories.
- **Versions are grouped**: `Kipfel_1.1.1`, `Kipfel_1.2.0` and `Zips/Kipfel v1.1.1.zip` become one asset.
- **BOOTH links are found** in folder names (`4460917 avatargimmick…`) and in readme / `.url` files,
  skipping dependencies such as lilToon.
- **Cover images** (`main.png`) become the preview.

Nothing is added until you accept it on the review screen. Re-scanning only reports what is new.

<p align="center"><img alt="Scan & Review screen with drafts" src="docs/screenshots/review.png" width="800"></p>

### 🛒 BOOTH import, on demand
Paste a BOOTH link and press **Fetch from BOOTH**: pick a preview image and copy the item name, shop,
category, tags and the avatars it supports. Works in bulk for scanner drafts too.

<p align="center"><img alt="Fetch from BOOTH panel" src="docs/screenshots/booth-import.png" width="800"></p>

### 👤 Avatar-first
Mark which avatars an outfit, hair or accessory fits — by hand, in bulk, or from the item's
"対応アバター" list on BOOTH. Each avatar gets a page with everything compatible, grouped by category.

<p align="center"><img alt="Avatar page with compatible assets" src="docs/screenshots/avatar-page.png" width="800"></p>

### ✨ Comfortable to use
- Drop or paste (Ctrl+V) an image to set a preview
- Multi-select to set a category, add a tag or mark compatibility for many assets at once
- One asset can track several folders, archives and versions, each with an *on disk / missing* status
- **Open folder** jumps straight to the files in Explorer
- Dark, light or system theme

## Your files stay yours

| | |
|---|---|
| **Local-first** | Everything is stored in a SQLite file in `data/`. No account, no cloud. |
| **Read-only on your library** | The app never moves, renames or deletes your asset files. Deleting or ignoring an entry only removes the library record and its preview copy. |
| **Private folders are skipped** | Folders in the *Never scan* list (by default `AvatarPass`) are never opened. |
| **BOOTH only when you ask** | booth.pm is contacted only when you press a fetch button, at most once per second; items are cached for a week. Preview images are downloaded, never hotlinked. |
| **Backups before upgrades** | Before applying a database migration, the backend copies `app.db` to `data/backups/`. |

## How it works

```mermaid
flowchart LR
    Browser["Browser<br/>localhost:3000"] -->|REST / JSON| API["Go API<br/>localhost:8080"]
    subgraph Frontend
      Browser
    end
    API --> DB[("SQLite<br/>data/app.db")]
    API --> Previews["Preview images<br/>data/previews/"]
    API -. "read only" .-> Library["Your asset folders<br/>e.g. D:\VRChat Assets"]
    API -. "on request" .-> BOOTH["booth.pm"]
```

The scanner turns folders into drafts you confirm:

```mermaid
flowchart LR
    A["Folders & zips<br/>on disk"] --> B["Group by name<br/>(width, case, version ignored)"]
    B --> C["Hints:<br/>category · BOOTH id ·<br/>compatible avatars · cover image"]
    C --> D["Drafts"]
    D -->|Accept| E["Library"]
    D -->|Ignore| F["Skipped on<br/>next scans"]
    D -->|Fetch BOOTH info| D
```

**Stack** — Next.js 16 (App Router), TypeScript, Tailwind CSS 4, shadcn/ui (Radix) and lucide icons on the
frontend; a Go standard-library HTTP server with the pure-Go `modernc.org/sqlite` driver and plain SQL
migrations on the backend.

## Getting started

### Just want to use it?

Download `VRChatAssetManager-<version>.zip`, unzip it anywhere, and double-click
**VRChatAssetManager.exe**. The app opens in your browser at http://127.0.0.1:47380 and stays in the
system tray (click to reopen, right-click → Quit). Your library is kept in the `data` folder next to the exe. No Go, Node.js or
terminal needed. The `README.txt` inside the zip covers the rest.

To make that zip yourself (needs Go and Node.js):

```powershell
.\scripts\build-release.ps1 -Version 1.0.0   # -> dist\VRChatAssetManager-1.0.0.zip
```

Or let GitHub do it: pushing a tag such as `v1.0.0` runs
[.github/workflows/release.yml](.github/workflows/release.yml), which publishes a Release with the zip.
Running apps show an "Update" button when a newer release exists.

The rest of this section is for running from source.

### Requirements

- [Go](https://go.dev/dl/) 1.25 or newer
- [Node.js](https://nodejs.org/) 20.9 or newer

### Run it

Start the backend (it creates `data/app.db` and applies migrations on first run):

```bash
cd backend
go run .
```

In a second terminal, start the frontend:

```bash
cd frontend
npm install
npm run dev
```

Open **http://localhost:3000**.

> The frontend is a static export (`output: "export"`), so `npm start` is not available; for everyday use
> build the release exe above.

### First steps

1. **Scan & Review** (sidebar) → **Settings** → add your asset folder and check the
   *folder name → category* table, then **Save**.
2. **Scan now**, look through the drafts, and **Accept** the ones you want — in bulk if you like.
3. Select drafts that have a BOOTH link and press **Fetch BOOTH info** to fill names and previews.
4. Open your avatars once with **Fetch from BOOTH** too: their Japanese names (e.g. キプフェル) help
   the app recognise which items fit them.

Prefer to start by hand? **Add Asset** in the header works without any scanning.

### Configuration

| Variable | Where | Default |
|---|---|---|
| `PORT` | backend | `8080` (release exe: `47380`) |
| `HOST` | backend | `127.0.0.1` (local only) |
| `DATA_DIR` | backend | `data/` in the repository (release exe: `data\` next to the exe) |
| `DB_PATH` | backend | `<DATA_DIR>/app.db` |
| `PREVIEWS_DIR` | backend | `<DATA_DIR>/previews` |
| `NO_BROWSER` | release exe | unset; set it to skip opening the browser |
| `NEXT_PUBLIC_API_URL` | `frontend/.env.local` | `http://localhost:8080` |

The backend accepts browser requests from `http://localhost:3000`. The release exe serves the UI itself,
so there the browser talks to a single origin.

## Project structure

```text
├── backend/
│   ├── main.go              # HTTP server, routes, startup backup + migrations
│   ├── migrations/          # Plain SQL migrations, embedded in the binary
│   ├── internal/
│   │   ├── asset/           # Assets, files & versions, compatibility, previews, bulk edit
│   │   ├── category/        # Two-level category tree
│   │   ├── scanner/         # Folder scanner, name matching, drafts
│   │   ├── booth/           # BOOTH lookup, cache, suggestions
│   │   ├── database/        # SQLite connection, migrations, backups
│   │   ├── desktop/         # Tray icon, folder picker, message boxes (Windows)
│   │   └── web/             # Serves the embedded UI in the release build
│   └── cmd/                 # seed (sample data) and verify (end-to-end checks)
├── frontend/
│   ├── app/                 # Library, asset, avatar, review and category pages
│   ├── components/          # Cards, drawer, forms, scanner and BOOTH panels
│   │   └── ui/              # shadcn/ui components
│   └── lib/                 # API client and helpers
├── data/                    # Your database, previews and backups (git-ignored)
├── docs/                    # API reference and screenshots
├── scripts/                 # build-release.ps1, the readme shipped in the zip, icon/
└── PROJECT_SPEC.md          # Specification and milestone history
```

## Development

```bash
# Backend tests (asset, category, scanner, BOOTH and database packages)
cd backend && go test ./...

# Frontend checks
cd frontend && npx tsc --noEmit && npm run lint && npm run build

# Sample data for an empty database
cd backend && go run ./cmd/seed
```

The REST API is documented in [docs/API.md](docs/API.md). Design decisions and the milestone history live in
[PROJECT_SPEC.md](PROJECT_SPEC.md).

## Roadmap

Done: asset CRUD, previews, search and tags, category tree, files & versions, avatar compatibility,
folder scanner with review, BOOTH import, refreshed UI with drawer, bulk edit, avatar pages and themes,
and a portable one-file release for non-developers (tray app, first-run welcome, update notice).

Ideas for later:

- **Collections** — named groups such as "Halloween" or "Avatar build #1"
- **Avatar builds** — save a full look: avatar + hair + outfit + accessories
- **Duplicate detection** — by BOOTH link or file hash
- **Optional Google Drive backup** of the database

---

<sub>Not affiliated with VRChat Inc. or pixiv Inc. (BOOTH). Item names and images belong to their creators.</sub>
