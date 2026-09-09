"use client";

import React, {
  useState,
  useEffect,
  useCallback,
  useTransition,
  Suspense,
  useRef,
} from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { Header } from "@/components/Header";
import { Sidebar } from "@/components/Sidebar";
import { AssetGrid } from "@/components/AssetGrid";
import {
  FilterToolbar,
  SortOption,
  LocalStatusOption,
} from "@/components/FilterToolbar";
import {
  Asset,
  Category,
  Tag,
  getAssets,
  getCategories,
  getTags,
  toggleAssetFavorite,
  checkBackendHealth,
  AssetFilterParams,
} from "@/lib/api";

function LibraryView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  // State
  const [categories, setCategories] = useState<Category[]>([]);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [assets, setAssets] = useState<Asset[]>([]);

  // Filter States initialized from URL
  const [selectedCategory, setSelectedCategory] = useState<string>(
    searchParams.get("category") || "all"
  );
  const [searchQuery, setSearchQuery] = useState<string>(
    searchParams.get("search") || ""
  );
  const [debouncedSearch, setDebouncedSearch] = useState<string>(
    searchParams.get("search") || ""
  );
  const [selectedTags, setSelectedTags] = useState<string[]>(() => {
    const raw = searchParams.get("tags");
    return raw ? raw.split(",").map((t) => t.trim()).filter(Boolean) : [];
  });
  const [isFavoriteOnly, setIsFavoriteOnly] = useState<boolean>(
    searchParams.get("favorite") === "true"
  );
  const [hasPreview, setHasPreview] = useState<boolean>(
    searchParams.get("has_preview") === "true"
  );
  const [hasBooth, setHasBooth] = useState<boolean>(
    searchParams.get("has_booth") === "true"
  );
  const [localStatus, setLocalStatus] = useState<LocalStatusOption>(() => {
    const s = searchParams.get("local_status");
    if (s === "available" || s === "missing" || s === "not_specified") {
      return s;
    }
    return "all";
  });
  const [sort, setSort] = useState<SortOption>(() => {
    const s = searchParams.get("sort");
    if (s === "updated" || s === "name_asc" || s === "name_desc") {
      return s;
    }
    return "recent";
  });

  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isCategoriesLoading, setIsCategoriesLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState<boolean>(true);

  const [, startTransition] = useTransition();
  const isFirstMount = useRef(true);

  // 1. Debounce search query input (250ms)
  useEffect(() => {
    const timer = setTimeout(() => {
      startTransition(() => {
        setDebouncedSearch(searchQuery);
      });
    }, 250);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  // 2. Synchronize filters to URL query string
  useEffect(() => {
    if (isFirstMount.current) {
      isFirstMount.current = false;
      return;
    }

    const params = new URLSearchParams();

    if (selectedCategory && selectedCategory !== "all") {
      params.set("category", selectedCategory);
    }
    if (debouncedSearch.trim()) {
      params.set("search", debouncedSearch.trim());
    }
    if (selectedTags.length > 0) {
      params.set("tags", selectedTags.join(","));
    }
    if (isFavoriteOnly) {
      params.set("favorite", "true");
    }
    if (hasPreview) {
      params.set("has_preview", "true");
    }
    if (hasBooth) {
      params.set("has_booth", "true");
    }
    if (localStatus !== "all") {
      params.set("local_status", localStatus);
    }
    if (sort !== "recent") {
      params.set("sort", sort);
    }

    const qs = params.toString();
    const target = qs ? `${pathname}?${qs}` : pathname;
    router.replace(target, { scroll: false });
  }, [
    selectedCategory,
    debouncedSearch,
    selectedTags,
    isFavoriteOnly,
    hasPreview,
    hasBooth,
    localStatus,
    sort,
    pathname,
    router,
  ]);

  // 3. Load categories and tags once
  const loadInitialMetadata = useCallback(async () => {
    setIsCategoriesLoading(true);
    try {
      const [cats, tags] = await Promise.all([getCategories(), getTags()]);
      setCategories(cats);
      setAvailableTags(tags.map((t: Tag) => t.name));
      setIsConnected(true);
    } catch {
      setIsConnected(false);
    } finally {
      setIsCategoriesLoading(false);
    }
  }, []);

  useEffect(() => {
    loadInitialMetadata();
  }, [loadInitialMetadata]);

  // 4. Fetch assets based on active filters
  const loadAssets = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const filterParams: AssetFilterParams = {
        category: selectedCategory !== "all" ? selectedCategory : undefined,
        search: debouncedSearch.trim() || undefined,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        favorite: isFavoriteOnly ? true : undefined,
        has_preview: hasPreview ? true : undefined,
        has_booth: hasBooth ? true : undefined,
        local_status: localStatus !== "all" ? localStatus : undefined,
        sort: sort !== "recent" ? sort : undefined,
      };

      const data = await getAssets(filterParams);
      setAssets(data);
      setIsConnected(true);
    } catch (err: unknown) {
      setIsConnected(false);
      if (err instanceof Error) {
        setError(err.message || "Failed to load assets from server");
      } else {
        setError("Unknown error while communicating with backend");
      }
    } finally {
      setIsLoading(false);
    }
  }, [
    selectedCategory,
    debouncedSearch,
    selectedTags,
    isFavoriteOnly,
    hasPreview,
    hasBooth,
    localStatus,
    sort,
  ]);

  useEffect(() => {
    loadAssets();
  }, [loadAssets]);

  // 5. Backend health check poll
  useEffect(() => {
    const interval = setInterval(async () => {
      const healthy = await checkBackendHealth();
      setIsConnected(healthy);
    }, 15000);

    return () => clearInterval(interval);
  }, []);

  // Tag filter handlers
  const handleToggleTag = (tag: string) => {
    setSelectedTags((prev) =>
      prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]
    );
  };

  const handleRemoveTag = (tag: string) => {
    setSelectedTags((prev) => prev.filter((t) => t !== tag));
  };

  const handleClearTags = () => {
    setSelectedTags([]);
  };

  // Reset all filters
  const handleClearAllFilters = () => {
    setSearchQuery("");
    setDebouncedSearch("");
    setSelectedCategory("all");
    setSelectedTags([]);
    setIsFavoriteOnly(false);
    setHasPreview(false);
    setHasBooth(false);
    setLocalStatus("all");
    setSort("recent");
  };

  // Optimistic favorite toggle
  const handleToggleFavorite = async (assetId: number, nextFav: boolean) => {
    setAssets((prev) =>
      prev.map((a) => (a.id === assetId ? { ...a, is_favorite: nextFav } : a))
    );

    try {
      await toggleAssetFavorite(assetId, nextFav);
    } catch (err) {
      console.error("Failed to toggle favorite:", err);
      // Revert on error
      setAssets((prev) =>
        prev.map((a) => (a.id === assetId ? { ...a, is_favorite: !nextFav } : a))
      );
    }
  };

  const handleRetry = () => {
    loadInitialMetadata();
    loadAssets();
  };

  // Computed properties
  const hasActiveFilters = Boolean(
    debouncedSearch.trim() ||
      selectedCategory !== "all" ||
      selectedTags.length > 0 ||
      isFavoriteOnly ||
      hasPreview ||
      hasBooth ||
      localStatus !== "all"
  );

  const favoriteCount = assets.filter((a) => a.is_favorite).length;

  return (
    <div className="min-h-screen flex flex-col bg-neutral-950 text-neutral-100 selection:bg-cyan-500/20 selection:text-cyan-200">
      {/* Top Navigation Bar */}
      <Header
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        isConnected={isConnected}
      />

      {/* Main Layout Area */}
      <div className="flex-1 flex flex-col md:flex-row">
        {/* Left Categories & Tag Filters Sidebar */}
        <Sidebar
          categories={categories}
          selectedCategory={selectedCategory}
          onSelectCategory={(cat) => setSelectedCategory(cat)}
          isFavoriteOnly={isFavoriteOnly}
          onToggleFavoriteOnly={() => setIsFavoriteOnly((prev) => !prev)}
          favoriteCount={favoriteCount}
          availableTags={availableTags}
          selectedTags={selectedTags}
          onToggleTag={handleToggleTag}
          onClearTags={handleClearTags}
          isLoading={isCategoriesLoading}
          totalAssetsCount={assets.length}
        />

        {/* Center/Right Asset Browsing View */}
        <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl">
          {/* Header & Title */}
          <div className="mb-2 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div>
              <h2 className="text-xl font-bold tracking-tight text-white capitalize flex items-center gap-2">
                {isFavoriteOnly && <span className="text-rose-500">♥</span>}
                {isFavoriteOnly
                  ? "Favorite Assets"
                  : selectedCategory === "all"
                  ? "All Assets"
                  : selectedCategory}
              </h2>
              <p className="text-xs text-neutral-400 mt-0.5">
                {isLoading ? (
                  "Loading asset library..."
                ) : (
                  <>
                    Showing <span className="font-semibold text-white">{assets.length}</span>{" "}
                    {assets.length === 1 ? "asset" : "assets"}
                    {debouncedSearch && (
                      <>
                        {" "}matching &ldquo;
                        <span className="text-cyan-400 font-mono">
                          {debouncedSearch}
                        </span>
                        &rdquo;
                      </>
                    )}
                  </>
                )}
              </p>
            </div>
          </div>

          {/* Filter & Sort Toolbar */}
          <FilterToolbar
            sort={sort}
            onSortChange={setSort}
            hasPreview={hasPreview}
            onToggleHasPreview={() => setHasPreview((prev) => !prev)}
            hasBooth={hasBooth}
            onToggleHasBooth={() => setHasBooth((prev) => !prev)}
            localStatus={localStatus}
            onLocalStatusChange={setLocalStatus}
            searchQuery={debouncedSearch}
            onClearSearch={() => {
              setSearchQuery("");
              setDebouncedSearch("");
            }}
            selectedTags={selectedTags}
            onRemoveTag={handleRemoveTag}
            isFavoriteOnly={isFavoriteOnly}
            onToggleFavoriteOnly={() => setIsFavoriteOnly((prev) => !prev)}
            totalCount={assets.length}
            hasActiveFilters={hasActiveFilters}
            onClearAllFilters={handleClearAllFilters}
          />

          {/* Asset Grid */}
          <AssetGrid
            assets={assets}
            isLoading={isLoading}
            error={error}
            onRetry={handleRetry}
            searchQuery={debouncedSearch}
            selectedCategory={selectedCategory}
            isFavoriteOnly={isFavoriteOnly}
            hasActiveFilters={hasActiveFilters}
            onClearFilters={handleClearAllFilters}
            onToggleFavorite={handleToggleFavorite}
          />
        </main>
      </div>
    </div>
  );
}

export default function Home() {
  return (
    <Suspense
      fallback={
        <div className="min-h-screen bg-neutral-950 flex items-center justify-center text-neutral-400 font-mono text-xs">
          Loading library...
        </div>
      }
    >
      <LibraryView />
    </Suspense>
  );
}
