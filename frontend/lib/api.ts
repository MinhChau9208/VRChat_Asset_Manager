// API client helper for VRChat Asset Manager

export interface Category {
  id: number;
  name: string;
  parent_id: number | null;
  sort_order: number;
  created_at: string;
}

export interface CategoryInput {
  name: string;
  parent_id?: number | null;
  sort_order?: number;
}

export type AssetStatusValue = "active" | "draft";

export interface AssetFile {
  id: number;
  asset_id: number;
  path: string;
  kind: "folder" | "archive" | "unitypackage" | "file";
  version: string;
  exists: boolean;
  created_at: string;
}

export interface CompatAvatar {
  avatar_asset_id: number | null;
  avatar_name: string;
}

export interface CategoryInfo {
  id: number;
  name: string;
}

export interface Asset {
  id: number;
  name: string;
  category_id: number | null;
  category?: CategoryInfo;
  author: string;
  booth_url: string;
  local_path: string;
  preview_path: string;
  description: string;
  tags: string[];
  is_favorite?: boolean;
  status?: AssetStatusValue;
  local_file_exists?: boolean;
  /** Only present on single-asset responses. */
  files?: AssetFile[];
  /** Only present on single-asset responses. */
  compatible_avatars?: CompatAvatar[];
  created_at: string;
  updated_at: string;
}

export interface AssetFilterParams {
  category?: string;
  search?: string;
  tag?: string;
  tags?: string[];
  favorite?: boolean;
  has_preview?: boolean;
  has_booth?: boolean;
  local_status?: string;
  sort?: string;
  status?: "active" | "draft" | "all";
  compatible_with?: number;
}

export interface Tag {
  id: number;
  name: string;
  created_at: string;
}

export interface CreateAssetInput {
  name: string;
  category_id?: number | null;
  author?: string;
  booth_url?: string;
  local_path?: string;
  preview_path?: string;
  description?: string;
  tags?: string[];
  is_favorite?: boolean;
  status?: AssetStatusValue;
  compatible_avatars?: CompatAvatar[];
}

export interface UpdateAssetInput {
  name: string;
  category_id?: number | null;
  author?: string;
  booth_url?: string;
  local_path?: string;
  preview_path?: string;
  description?: string;
  tags?: string[];
  is_favorite?: boolean;
  status?: AssetStatusValue;
  compatible_avatars?: CompatAvatar[];
}

const getApiBaseUrl = (): string => {
  return process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
};

/**
 * Fetch all categories from the backend.
 */
export async function getCategories(): Promise<Category[]> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/categories`, {
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to fetch categories`);
  }

  return res.json();
}

/**
 * Fetch assets with optional filters (category, search, tag, tags, favorite, etc.).
 */
export async function getAssets(filters?: AssetFilterParams): Promise<Asset[]> {
  const baseUrl = getApiBaseUrl();
  const params = new URLSearchParams();

  if (filters?.search && filters.search.trim()) {
    params.set("search", filters.search.trim());
  }

  if (filters?.category && filters.category.trim() && filters.category !== "all") {
    params.set("category", filters.category.trim());
  }

  if (filters?.tag && filters.tag.trim()) {
    params.set("tag", filters.tag.trim());
  }

  if (filters?.tags && filters.tags.length > 0) {
    params.set("tags", filters.tags.join(","));
  }

  if (filters?.favorite !== undefined) {
    params.set("favorite", String(filters.favorite));
  }

  if (filters?.has_preview !== undefined) {
    params.set("has_preview", String(filters.has_preview));
  }

  if (filters?.has_booth !== undefined) {
    params.set("has_booth", String(filters.has_booth));
  }

  if (filters?.local_status && filters.local_status !== "all") {
    params.set("local_status", filters.local_status.trim());
  }

  if (filters?.sort && filters.sort.trim()) {
    params.set("sort", filters.sort.trim());
  }

  if (filters?.status && filters.status !== "active") {
    params.set("status", filters.status);
  }

  if (filters?.compatible_with !== undefined) {
    params.set("compatible_with", String(filters.compatible_with));
  }

  const queryString = params.toString();
  const url = `${baseUrl}/api/assets${queryString ? `?${queryString}` : ""}`;

  const res = await fetch(url, {
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to fetch assets`);
  }

  return res.json();
}

/**
 * Fetch a single asset by ID.
 */
export async function getAssetByID(id: string | number): Promise<Asset> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}`, {
    cache: "no-store",
  });

  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("Asset not found");
    }
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to fetch asset`);
  }

  return res.json();
}

export interface AssetStatus {
  exists: boolean;
}

/**
 * Check if the asset's local file or directory exists on disk.
 */
export async function getAssetStatus(id: string | number): Promise<AssetStatus> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}/status`, {
    cache: "no-store",
  });

  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("Asset not found");
    }
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to check asset status`);
  }

  return res.json();
}

/**
 * Construct the stable URL for retrieving an asset's preview image.
 */
export function getAssetPreviewUrl(id: string | number, updatedAt?: string): string {
  const baseUrl = getApiBaseUrl();
  const url = `${baseUrl}/api/assets/${id}/preview`;
  if (updatedAt) {
    return `${url}?t=${encodeURIComponent(updatedAt)}`;
  }
  return url;
}

/**
 * Upload a preview image (JPEG, PNG, WebP) for an asset.
 */
export async function uploadAssetPreview(id: string | number, file: File): Promise<Asset> {
  const baseUrl = getApiBaseUrl();
  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(`${baseUrl}/api/assets/${id}/preview`, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("Asset not found");
    }
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to upload preview`);
  }

  return res.json();
}

/**
 * Delete an asset's preview image.
 */
export async function deleteAssetPreview(id: string | number): Promise<{ message: string }> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}/preview`, {
    method: "DELETE",
  });

  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("Asset not found");
    }
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to delete preview`);
  }

  return res.json();
}

/**
 * Request backend to open the asset folder or file in the OS explorer.
 */
export async function openAssetFolder(id: string | number): Promise<{ status: string }> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}/open-folder`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to open folder`);
  }

  return res.json();
}

/**
 * Quick check to verify backend health.
 */
export async function checkBackendHealth(): Promise<boolean> {
  try {
    const baseUrl = getApiBaseUrl();
    const res = await fetch(`${baseUrl}/health`, {
      cache: "no-store",
    });
    return res.ok;
  } catch {
    return false;
  }
}

/**
 * Create a new asset.
 */
export async function createAsset(input: CreateAssetInput): Promise<Asset> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to create asset`);
  }

  return res.json();
}

/**
 * Update an existing asset.
 */
export async function updateAsset(
  id: string | number,
  input: UpdateAssetInput
): Promise<Asset> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(input),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to update asset`);
  }

  return res.json();
}

/**
 * Delete an existing asset.
 */
export async function deleteAsset(id: string | number): Promise<{ message: string }> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}`, {
    method: "DELETE",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to delete asset`);
  }

  return res.json();
}

/**
 * Fetch all existing tags.
 */
export async function getTags(): Promise<Tag[]> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/tags`, {
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to fetch tags`);
  }

  return res.json();
}

/**
 * Create a new tag (or return existing if duplicate).
 */
export async function createTag(name: string): Promise<Tag> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/tags`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ name }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to create tag`);
  }

  return res.json();
}

/**
 * Trigger native Windows folder browser picker via backend.
 * Returns the selected path, or empty string if cancelled or unsupported.
 */
export async function pickFolder(): Promise<string> {
  const baseUrl = getApiBaseUrl();
  try {
    const res = await fetch(`${baseUrl}/api/filesystem/pick-folder`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (!res.ok) {
      return "";
    }

    const data = await res.json().catch(() => ({ path: "" }));
    return data.path || "";
  } catch {
    return "";
  }
}

/**
 * Toggle or explicitly set favorite status for an asset.
 */
export async function toggleAssetFavorite(
  id: string | number,
  isFavorite?: boolean
): Promise<Asset> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/${id}/favorite`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: isFavorite !== undefined ? JSON.stringify({ is_favorite: isFavorite }) : "{}",
  });

  if (!res.ok) {
    if (res.status === 404) {
      throw new Error("Asset not found");
    }
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to toggle favorite`);
  }

  return res.json();
}

/**
 * Batch check local file existence for multiple assets.
 */
export async function getBatchAssetStatus(
  ids: number[]
): Promise<Record<string, boolean>> {
  if (!ids || ids.length === 0) {
    return {};
  }

  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/assets/batch-status`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ ids }),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to check batch status`);
  }

  const data = await res.json();
  return data.statuses || {};
}


export interface LibraryStats {
  total: number;
  favorites: number;
  by_category: Record<string, number>;
}

/**
 * Fetch unfiltered library counts (total, favorites, per category).
 */
export async function getLibraryStats(): Promise<LibraryStats> {
  const baseUrl = getApiBaseUrl();
  const res = await fetch(`${baseUrl}/api/stats`, {
    cache: "no-store",
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: Failed to fetch library stats`);
  }

  return res.json();
}

async function sendJSON<T>(method: string, path: string, body: unknown, fallback: string): Promise<T> {
  const res = await fetch(`${getApiBaseUrl()}${path}`, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({}));
    throw new Error(errorData.error || `HTTP ${res.status}: ${fallback}`);
  }

  return res.json();
}

/**
 * Create a category (optionally under a top-level parent).
 */
export function createCategory(input: CategoryInput): Promise<Category> {
  return sendJSON("POST", "/api/categories", input, "Failed to create category");
}

/**
 * Rename, re-parent or reorder a category. Omitting sort_order keeps it.
 */
export function updateCategory(id: number, input: CategoryInput): Promise<Category> {
  return sendJSON("PUT", `/api/categories/${id}`, input, "Failed to update category");
}

/**
 * Delete a category without subcategories; its assets become uncategorized.
 */
export function deleteCategory(id: number): Promise<{ message: string }> {
  return sendJSON("DELETE", `/api/categories/${id}`, undefined, "Failed to delete category");
}

/**
 * Link another file/folder (e.g. an archived zip or older version) to an asset.
 */
export function addAssetFile(
  assetId: number | string,
  input: { path: string; version?: string; kind?: AssetFile["kind"] }
): Promise<AssetFile> {
  return sendJSON("POST", `/api/assets/${assetId}/files`, input, "Failed to add file");
}

/**
 * Unlink a file from an asset. Nothing on disk is touched.
 */
export function deleteAssetFile(assetId: number | string, fileId: number): Promise<{ message: string }> {
  return sendJSON("DELETE", `/api/assets/${assetId}/files/${fileId}`, undefined, "Failed to remove file");
}

export interface CategoryNode extends Category {
  children: Category[];
}

/**
 * Group the flat, display-ordered category list into a two-level tree.
 */
export function buildCategoryTree(categories: Category[]): CategoryNode[] {
  const roots: CategoryNode[] = [];
  const byId = new Map<number, CategoryNode>();
  for (const c of categories) {
    if (c.parent_id === null) {
      const node = { ...c, children: [] };
      roots.push(node);
      byId.set(c.id, node);
    }
  }
  for (const c of categories) {
    if (c.parent_id !== null) {
      byId.get(c.parent_id)?.children.push(c);
    }
  }
  return roots;
}
