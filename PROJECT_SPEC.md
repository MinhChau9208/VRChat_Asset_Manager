# VRChat Asset Manager — Project Specification

## 1. Project goal

Build a **local-first web application** for managing a personal library of VRChat assets.

The application should make it easy to answer:

> "Mình đã mua những asset gì, chúng trông như thế nào, nằm ở đâu trên máy, và link BOOTH là gì?"

Typical assets:

- Avatars
- Hair
- Clothes / outfits
- Shoes
- Accessories
- Gimmicks
- Textures
- Materials
- Shaders
- Unity packages
- Other VRChat-related files

The application is primarily for **localhost / personal use**.

---

## 2. Core principles

### 2.1 Local-first

Local filesystem + local database are the primary source of truth.

Do NOT make Google Drive a required dependency.

Google Drive can be added later as optional backup/synchronization.

### 2.2 Metadata in database, files on disk

Do not store large asset files inside the database.

Database stores metadata such as:

- Name
- Category
- Author
- BOOTH URL
- Local path
- Preview path
- Tags
- Description
- Created/updated timestamps

Actual files remain on the filesystem.

### 2.3 MVP before automation

First make manual asset management work.

Do NOT start with:

- Google Drive integration
- BOOTH scraping
- Automatic image scraping
- Complex avatar-building logic
- Authentication
- Multi-user support

These are future features.

---

# 3. MVP

The first usable version should support:

1. Create an asset
2. Edit an asset
3. Delete an asset
4. Display asset preview
5. Store BOOTH URL
6. Store local filesystem path
7. Store category
8. Store tags
9. Search by name
10. Filter by category
11. Open BOOTH link
12. Open local asset folder
13. View asset details

A user should be able to go from:

    Add Asset
       ↓
    Enter metadata
       ↓
    Upload/select preview
       ↓
    Save
       ↓
    Asset appears in grid
       ↓
    Click asset
       ↓
    Open BOOTH / local folder

before adding any advanced features.

---

# 4. Recommended stack

## Frontend

- Next.js
- TypeScript
- Tailwind CSS
- shadcn/ui

## Backend

Preferred for this project:

- Go
- REST API
- JSON

Alternative for a very fast MVP:

- Next.js API routes / server actions

However, using Go is encouraged because this project can also be used to practice backend development.

## Database

Start with:

- SQLite
- Prisma if using a TypeScript backend

If using a Go backend, use a lightweight SQLite driver and SQL migrations.

Do not start with PostgreSQL unless there is a specific reason.

## Development

Recommended editors:

- VS Code
- Antigravity / another coding agent

The project should remain understandable and runnable without depending on the coding agent.

---

# 5. Initial architecture

Preferred architecture:

    Browser
       |
       v
    Next.js frontend
       |
       | HTTP / REST
       v
    Go API
       |
       +------ SQLite
       |
       +------ Local asset filesystem
       |
       +------ Preview images

Example:

    localhost:3000  -> frontend
    localhost:8080  -> backend

The backend is responsible for:

- CRUD operations
- validation
- database access
- filesystem operations
- opening local paths when supported by the OS

The frontend is responsible for:

- UI
- filtering
- search
- forms
- asset cards
- asset details

---

# 6. Repository structure

Suggested structure:

    vrchat-asset-manager/
    |
    +-- frontend/
    |   +-- app/
    |   +-- components/
    |   +-- lib/
    |   +-- public/
    |   +-- package.json
    |
    +-- backend/
    |   +-- cmd/
    |   +-- internal/
    |   |   +-- asset/
    |   |   +-- category/
    |   |   +-- tag/
    |   |   +-- database/
    |   |   +-- filesystem/
    |   +-- migrations/
    |   +-- go.mod
    |   +-- main.go
    |
    +-- data/
    |   +-- app.db
    |   +-- previews/
    |
    +-- docs/
    |
    +-- .gitignore
    +-- README.md
    +-- docker-compose.yml

Important:

Do NOT commit:

- personal asset files
- preview images if they are copyrighted and should remain private
- local database containing personal metadata, unless intentionally desired
- secrets

---

# 7. Data model

Start small.

## Asset

Suggested fields:

    id
    name
    category_id
    author
    booth_url
    local_path
    preview_path
    description
    created_at
    updated_at

## Category

    id
    name

Initial categories:

    Avatar
    Hair
    Clothes
    Shoes
    Accessory
    Gimmick
    Texture
    Material
    Shader
    Other

## Tag

    id
    name

Examples:

    anime
    long-hair
    short-hair
    dress
    gothic
    cute
    physbone
    poiyomi
    quest
    feminine
    masculine

## AssetTag

Many-to-many relation:

    asset_id
    tag_id

---

# 8. Future data model

Do not implement these in the MVP unless necessary.

Possible future entities:

    Author
    Collection
    Avatar
    Compatibility
    Purchase
    AssetVersion
    AssetFile
    AvatarBuild

Possible future relationships:

    Outfit -> compatible with Avatar
    Hair -> compatible with Avatar
    Accessory -> compatible with Avatar
    Gimmick -> compatible with Avatar

---

# 9. UI plan

## Main layout

    +------------------------------------------------------+
    | VRChat Asset Manager                 [Add Asset]      |
    +------------------------------------------------------+
    | Search assets...                                     |
    +------------------+-----------------------------------+
    | Categories       |                                   |
    |                  |   [Asset] [Asset] [Asset]        |
    | All              |                                   |
    | Avatars          |   [Asset] [Asset] [Asset]        |
    | Hair             |                                   |
    | Clothes          |   [Asset] [Asset] [Asset]        |
    | Shoes            |                                   |
    | Accessories      |                                   |
    | Gimmicks         |                                   |
    | Materials        |                                   |
    | Shaders          |                                   |
    | Other            |                                   |
    +------------------+-----------------------------------+

## Asset card

Each card should show:

- Preview
- Asset name
- Category
- Optional author
- Favorite status (future)
- Small tag indicators (optional)

Example:

    +----------------------+
    |                      |
    |      PREVIEW         |
    |                      |
    +----------------------+
    | Cute Anime Hair      |
    | Hair                 |
    | @ArtistName          |
    | [anime] [long]       |
    +----------------------+

## Asset details

    Preview

    Cute Anime Hair

    Category: Hair
    Author: ArtistName

    Tags:
    [anime] [long] [ponytail]

    Local path:
    D:\VRChatAssets\Hair\CuteHair

    BOOTH:
    https://booth.pm/...

    [Open Folder]
    [Open BOOTH]
    [Edit]
    [Delete]

---

# 10. Important local filesystem behavior

The app should treat `local_path` as a reference to an existing file/folder.

Example:

    D:\VRChatAssets\Hair\CuteHair

Do not automatically copy the entire asset into the web app.

The application should support:

- Checking whether local_path exists
- Showing "Available" / "Missing"
- Opening the path using the local OS when possible

Example:

    Local files
    [✓ Available]

or:

    Local files
    [✕ Missing]

This will become useful if files are moved.

---

# 11. Preview image handling

For MVP:

- User selects/uploads a preview image
- Application stores the preview in a local previews directory
- Database stores the preview path

Example:

    data/previews/
        asset-001.webp
        asset-002.jpg
        asset-003.png

Recommended image behavior:

- Accept JPG / PNG / WebP
- Generate a predictable filename using the asset ID
- Validate file type
- Limit upload size
- Prefer WebP for generated thumbnails later

Do not rely on BOOTH images being permanently available.

BOOTH URL and preview image should be independent.

---

# 12. BOOTH behavior

Store the BOOTH URL exactly as provided.

Example:

    https://booth.pm/en/items/123456

The frontend should render:

    [Open BOOTH]

Clicking it opens the external URL in a new browser tab.

Future feature:

- Import BOOTH metadata
- Suggest title
- Suggest preview image
- Suggest creator

Do not make scraping a requirement for the MVP.

---

# 13. Search and filtering

MVP search:

- Search by asset name
- Search by author
- Search by tag

Filters:

- Category
- Tags

Future filters:

- Has local files
- Missing local files
- Has BOOTH URL
- Favorites
- Recently added

Search should be case-insensitive.

Example:

    Search: "gothic"

could return:

    Gothic Dress
    Gothic Boots
    Gothic Choker

---

# 14. Recommended API

Keep API simple.

## Assets

    GET    /api/assets
    GET    /api/assets/:id
    POST   /api/assets
    PUT    /api/assets/:id
    DELETE /api/assets/:id

## Categories

    GET  /api/categories
    POST /api/categories

## Tags

    GET  /api/tags
    POST /api/tags

## Preview

    POST /api/assets/:id/preview

## Filesystem

    GET  /api/assets/:id/status
    POST /api/assets/:id/open-folder

Do not over-engineer the API.

---

# 15. Development milestones

## Milestone 0 — Project setup

Goal:

Both frontend and backend run locally.

Tasks:

- Create repository
- Create Next.js frontend
- Create Go backend
- Configure CORS if required
- Create health endpoint
- Connect frontend to backend
- Add README

Success condition:

    http://localhost:3000

can successfully call:

    http://localhost:8080/health

---

## Milestone 1 — Database

Tasks:

- Create SQLite database
- Create migrations
- Create Asset table
- Create Category table
- Create Tag table
- Create AssetTag table
- Seed default categories

Success condition:

A test asset can be stored and retrieved.

---

## Milestone 2 — Asset CRUD

Implement:

- Create
- Read
- Update
- Delete

Success condition:

A user can fully manage assets through the API.

---

## Milestone 3 — Asset grid UI

Create:

- Sidebar
- Search bar
- Asset cards
- Category filter
- Empty state
- Loading state
- Error state

Success condition:

The UI can browse assets comfortably.

---

## Milestone 4 — Asset details

Create:

- Detail page/modal
- Preview
- BOOTH link
- Local path
- Tags
- Edit form
- Delete action

Success condition:

A user can inspect one asset and reach its external/local resources.

---

## Milestone 5 — Preview upload

Implement:

- Upload preview
- Validate image type
- Store local preview
- Serve preview from backend
- Display preview on cards

Success condition:

An asset remains visually identifiable without opening BOOTH.

---

## Milestone 6 — Search + tags

Implement:

- Tag creation
- Tag assignment
- Search
- Category filtering
- Tag filtering

Success condition:

The library remains usable with hundreds of assets.

---

## Milestone 6.5 — Cleanup (done)

- Editing an asset no longer wipes its preview (`preview_path` is optional on update)
- Frontend lint clean
- Sidebar counts come from `GET /api/stats` (unfiltered) instead of the filtered list

---

## Milestone 7 — Data model v2

The MVP (M0–M6) is done. Before scanning real folders, the data model must
describe how assets actually exist on disk and how they relate to avatars.

### Category tree

Categories get an optional `parent_id` (two levels) and a `sort_order`.
Categories can be created, renamed, reordered and deleted from the UI
(`POST/PUT/DELETE /api/categories`).

Default tree (child ← example user folder, BOOTH category):

    Avatar                ← Models        (3Dキャラクター)
    Outfit
       Clothes            ← Clothes       (3D衣装)
       Shoes
    Hair                  ← Hair
    Accessory
       Accessory          ← Accessory     (3D装飾品)
       Ears & Tail        ← Ears
    Face
       Eyes               ← Eyes          (3Dテクスチャ)
       Expression         ← Facials
       Makeup             ← MakeUps
    Gimmick
       Gimmick            ← Gimmick
       Prop               ← PianoGimick   (3D小道具)
    Animation                             (3Dモーション・アニメーション)
    Texture & Material                    (3Dテクスチャ, 3Dマテリアル)
    Tool & Shader         ← community     (3Dツール・システム)
    World                                 (3D環境・ワールド)
    Audio                 ← Audio
    Other                 ← UnOrganized, loose items at the library root

Assets can sit on a parent ("Outfit") or a child ("Clothes"); filtering by a
parent includes its children. Names are unique across the tree.

Existing categories are migrated into this tree; no asset loses its category
(Texture → Texture & Material, Shader → Tool & Shader, Material merged into
Texture & Material). The backend backs up `app.db` to `data/backups/` before
applying any new migration.

### Asset files & versions

One asset can point to several files/folders on disk:

    asset_files
        id
        asset_id
        path            (unique)
        kind            folder | archive | unitypackage
        version         optional label, e.g. 1.2.0
        created_at

`assets.local_path` stays as the "primary" location for compatibility.
Example: `Models/Kipfel_1.1.1`, `Models/Kipfel_1.2.0` and
`Models/Kipfel_1.2.0.zip` are three files of one asset.

### Avatar compatibility

    asset_compat
        asset_id
        avatar_asset_id   nullable, an owned asset in the Avatar category
        avatar_name       used when the avatar is not owned (e.g. "Manuka")

An avatar's detail page lists everything compatible with it.

### Draft status

`assets.status`: `active` (default) | `draft`. Scanner-created assets start as
drafts and are hidden from the main library until confirmed.

Success condition:

Existing data migrates cleanly; an asset can have several files and
compatible avatars; categories are editable.

---

## Milestone 8 — Filesystem scanner

Only after M7.

Settings: one or more library roots (e.g. `N:\Unity Materials`), an ignore
list (default includes `AvatarPass`, which may contain passwords — never read
it), archive folder names (default `LEGACY`, `Legacy`, `Zips`), and a
folder-name → category mapping.

Rules:

- Level 1 under a root = category folder (mapped, case-insensitive).
  Loose items at the root go to Inbox.
- Level 2 = one asset: a folder, or an archive (`.zip`, `.7z`,
  `.unitypackage`) with no extracted folder.
- Archives inside archive folders are attached to the matching extracted
  folder, searching all archive folders (a zip may sit in another
  category's LEGACY). Matching normalizes names: NFKC (＆ → &), case,
  `_` / `-` / space as one separator, version suffix stripped.
- Version suffixes (`_v1.2`, `ver1.02`, `Ver_1.3`, `1.1.1`) group folders of
  the same asset into versions.
- Mojibake Shift-JIS names (e.g. `ìRïcâfâé…`) get a decoded name suggestion.
  (Deferred: needs a Shift-JIS decoder dependency; no asset-level name in the
  current library is affected.)
- BOOTH id candidates, in order of confidence:
  1. 6–8 digit number in the folder/file name (e.g. `4460917 avatargimmick …`)
  2. BOOTH links in `.url` / readme files inside the asset folder, excluding
     known dependencies (lilToon, BlendShare, Modular Avatar…); links to
     owned avatars become compatibility suggestions instead
- Preview candidate: an image at the top of the asset folder (`main.png`).

Every guess is a suggestion. The scanner creates drafts; a **Review screen**
lists new drafts, already-linked paths and assets whose files went missing,
and lets the user accept / edit / ignore, individually or in bulk.
Re-scanning only reports changes (paths are unique).

The scanner never moves, renames or deletes user files.

Success condition:

Scanning the real library produces one draft per asset (folder + zips +
versions grouped), with category filled and BOOTH id where reliable.

---

## Milestone 9 — BOOTH metadata import

On user action only (button "Fetch from BOOTH" in the form and Review screen):

    GET https://booth.pm/ja/items/{id}.json

Provides name, shop (→ author), category (→ category mapping), full-size
images, tags, variations (often name the target avatar) and description (often
a `対応アバター` section linking base avatars).

- Suggestions are shown and confirmed by the user before saving.
- The chosen image is downloaded into `data/previews/` (never hotlinked).
- At most 1 request per second; responses are cached locally.
- Adult items may require extra handling.

Success condition:

Paste a BOOTH URL → name, author, category, preview and compatibility are
suggested and saved in a few clicks.

---

## Milestone 10 — UI refresh

- shadcn/ui components and a consistent icon set (lucide) instead of emoji
- Square preview cards (BOOTH images are 1:1), adjustable card size, list view
- Category tree in the sidebar with counts
- Detail as a drawer, keeping grid position
- Preview in the create form: file picker, drag & drop, paste (Ctrl+V)
- Bulk select → set category / tags / compatibility
- Avatar page: everything compatible with an avatar
- Larger minimum text size

---

# 16. Future roadmap

Potential features:

## Collections

Example:

    My Favorites
    Halloween Assets
    Cute Avatar
    Avatar Build #01

## Favorites

Favorite assets should be one click away.

## Recently added

Show newest assets.

## Compatibility

Example:

    Hair A
    Compatible with:
        Manuka
        Moe
        Kikyo

## Avatar builds

Example:

    Cute Manuka

    Avatar: Manuka
    Hair: Cute Hair
    Outfit: Gothic Dress
    Shoes: Black Boots
    Accessory: Cat Ears
    Gimmick: Tail

## Duplicate detection

Possible detection methods:

- Same local path
- Same filename
- Same BOOTH URL
- File hash

## Google Drive backup

Only after local workflow is stable.

Google Drive should be treated as:

    optional backup/sync

not the primary database.

## BOOTH metadata import

Potential workflow:

    Paste BOOTH URL
          ↓
    Fetch metadata
          ↓
    Suggest title / creator / image
          ↓
    User confirms
          ↓
    Save

This should be optional.

---

# 17. Non-goals

Do not turn this into:

- A public marketplace
- A multi-user SaaS
- A replacement for Unity
- A cloud asset storage service
- A full VRChat avatar editor

The project is first and foremost a **personal asset catalog**.

---

# 18. Coding rules

When using an AI coding agent:

1. Implement one milestone at a time.
2. Do not rewrite working architecture without a reason.
3. Keep backend and frontend responsibilities separate.
4. Prefer simple code over abstraction-heavy code.
5. Add validation at API boundaries.
6. Add error handling.
7. Keep database migrations explicit.
8. Avoid introducing large dependencies for tiny features.
9. Keep secrets out of Git.
10. Update README when setup changes.

---

# 19. First implementation task

Start with only this:

### Task

Create the initial repository with:

    frontend/
    backend/
    data/
    docs/

Frontend:

- Next.js
- TypeScript
- Tailwind CSS
- Basic layout
- `/` page with a placeholder dashboard

Backend:

- Go
- `/health` endpoint
- CORS for local frontend development
- Clean project structure

Database:

- SQLite
- Initial migration system
- No complex schema yet

Documentation:

- README.md
- Explain how to run frontend and backend locally

### Definition of done

The developer can run:

    cd frontend
    npm install
    npm run dev

and separately:

    cd backend
    go run .

Then:

    GET http://localhost:8080/health

returns:

    {"status":"ok"}

and the frontend successfully displays:

    Backend: Connected

Do not implement asset CRUD yet.

---

# 20. Prompt for an AI coding agent

Use the following as the first prompt in VS Code / Antigravity:

> Read `PROJECT_SPEC.md` completely before changing code.
>
> We are building a local-first VRChat Asset Manager.
>
> Implement **Milestone 0 only**.
>
> Requirements:
>
> - Create a `frontend/` Next.js TypeScript application.
> - Create a `backend/` Go application.
> - Create a `/health` HTTP endpoint returning JSON `{ "status": "ok" }`.
> - Configure the frontend to call the backend health endpoint.
> - Display the backend connection state on the homepage.
> - Prepare `data/` and `docs/` directories.
> - Add a root README explaining setup and development commands.
> - Keep the architecture simple and maintainable.
> - Do NOT implement asset CRUD, Google Drive, BOOTH scraping, authentication, or advanced features yet.
>
> Before coding:
>
> 1. Inspect the repository.
> 2. Decide the minimal file structure needed.
> 3. Explain the files you intend to create/change.
> 4. Then implement Milestone 0.
>
> After implementation:
>
> - Run/build both frontend and backend if possible.
> - Fix any errors.
> - Summarize what changed.
> - Give the exact commands needed to run the project.

---

# 21. Suggested first git commits

Keep commits small.

    chore: initialize project structure

    feat: add go backend health endpoint

    feat: connect frontend to backend health endpoint

    docs: add project setup guide

Later:

    feat: add sqlite schema

    feat: add asset crud api

    feat: add asset grid

    feat: add asset details page

    feat: add preview upload

    feat: add search and tags

---

# 22. What success looks like

The final application should feel like:

    Steam Library
          +
    Pinterest-style asset previews
          +
    File Explorer
          +
    VRChat avatar inventory

The user should be able to open the application and immediately understand:

    What assets do I own?
    What does each asset look like?
    Where is it stored?
    Who made it?
    What is the BOOTH page?
    Is the local file still available?
    Which assets work together?

That is the long-term direction.

---

# 23. Immediate next action

Milestones 0–6.5 are done (see README "Milestone Status").

Continue with:

    Milestone 7  (data model v2)
        ↓
    Milestone 8  (scanner + review)
        ↓
    Milestone 9  (BOOTH import)
        ↓
    Milestone 10 (UI refresh)

One milestone at a time; every automated guess stays a suggestion the user confirms.

This keeps the project small enough to finish while still leaving a clean path toward a full VRChat asset library.
