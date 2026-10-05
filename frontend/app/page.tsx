"use client";

import React, {
  useState,
  useEffect,
  useTransition,
  Suspense,
  useRef,
} from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { Header } from "@/components/Header";
import { Sidebar } from "@/components/Sidebar";
import { AssetGrid } from "@/components/AssetGrid";
import { WelcomePanel } from "@/components/WelcomePanel";
import { AssetDetail } from "@/components/AssetDetail";
import { BulkActionBar } from "@/components/BulkActionBar";
import { CardSize, ViewMode } from "@/components/AssetCard";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useLocalStorage } from "@/lib/useLocalStorage";
import { useI18n } from "@/lib/i18n";
import { toast } from "sonner";
import { Heart, UserRound, X } from "lucide-react";
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
  getLibraryStats,
  LibraryStats,
  toggleAssetFavorite,
  checkBackendHealth,
  AssetFilterParams,
  BulkUpdateInput,
  bulkUpdateAssets,
} from "@/lib/api";

function LibraryView() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { t, categoryName } = useI18n();

  // State
  const [categories, setCategories] = useState<Category[]>([]);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [stats, setStats] = useState<LibraryStats | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [avatars, setAvatars] = useState<Asset[]>([]);

  // Display preferences, remembered per browser.
  const [view, setView] = useLocalStorage<ViewMode>("library.view", "grid", ["grid", "list"]);
  const [cardSize, setCardSize] = useLocalStorage<CardSize>("library.cardSize", "md", ["sm", "md", "lg"]);

  // Asset opened in the side drawer (kept in the URL as ?asset=ID).
  const [openAssetId, setOpenAssetId] = useState<number | null>(() => {
    const id = Number(searchParams.get("asset"));
    return id > 0 ? id : null;
  });
  const drawerChanged = useRef(false);

  // Select mode for bulk edits.
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [bulkBusy, setBulkBusy] = useState(false);

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
  // "Assets compatible with avatar X" (from an avatar's detail page)
  const [compatibleWith, setCompatibleWith] = useState<{ id: number; name: string } | null>(() => {
    const id = Number(searchParams.get("compatible_with"));
    return id > 0 ? { id, name: searchParams.get("for") || `#${id}` } : null;
  });
  const [sort, setSort] = useState<SortOption>(() => {
    const s = searchParams.get("sort");
    if (s === "updated" || s === "name_asc" || s === "name_desc") {
      return s;
    }
    return "recent";
  });

  const [loadedKey, setLoadedKey] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
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
    if (compatibleWith) {
      params.set("compatible_with", String(compatibleWith.id));
      params.set("for", compatibleWith.name);
    }
    if (openAssetId !== null) {
      params.set("asset", String(openAssetId));
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
    compatibleWith,
    openAssetId,
    pathname,
    router,
  ]);

  // 3. Load categories and tags (again on retry)
  useEffect(() => {
    let cancelled = false;
    Promise.all([getCategories(), getTags(), getLibraryStats(), getAssets({ category: "Avatar", sort: "name_asc" })])
      .then(([cats, tags, libraryStats, avatarList]) => {
        if (cancelled) return;
        setCategories(cats);
        setAvatars(avatarList);
        setAvailableTags(tags.map((t: Tag) => t.name));
        setStats(libraryStats);
        setIsConnected(true);
      })
      .catch(() => {
        if (!cancelled) setIsConnected(false);
      })
      .finally(() => {
        if (!cancelled) setIsCategoriesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [reloadToken]);

  // 4. Fetch assets based on active filters. Loading is derived from whether
  // the last completed request matches the current filters, so the effect
  // never has to set state synchronously.
  const filterParams: AssetFilterParams = {
    category: selectedCategory !== "all" ? selectedCategory : undefined,
    search: debouncedSearch.trim() || undefined,
    tags: selectedTags.length > 0 ? selectedTags : undefined,
    favorite: isFavoriteOnly ? true : undefined,
    has_preview: hasPreview ? true : undefined,
    has_booth: hasBooth ? true : undefined,
    local_status: localStatus !== "all" ? localStatus : undefined,
    sort: sort !== "recent" ? sort : undefined,
    compatible_with: compatibleWith?.id,
  };
  const filtersJson = JSON.stringify(filterParams);
  const requestKey = `${filtersJson}#${reloadToken}`;
  const isLoading = loadedKey !== requestKey;

  useEffect(() => {
    let cancelled = false;
    const params: AssetFilterParams = JSON.parse(filtersJson);

    getAssets(params)
      .then((data) => {
        if (cancelled) return;
        setAssets(data);
        setError(null);
        setIsConnected(true);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setIsConnected(false);
        if (err instanceof Error) {
          setError(err.message || t.library.loadFailed);
        } else {
          setError(t.library.unknownError);
        }
      })
      .finally(() => {
        if (!cancelled) setLoadedKey(requestKey);
      });

    return () => {
      cancelled = true;
    };
  }, [filtersJson, requestKey, t]);

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
    setCompatibleWith(null);
  };

  // Optimistic favorite toggle
  const handleToggleFavorite = async (assetId: number, nextFav: boolean) => {
    const adjustFavorites = (delta: number) =>
      setStats((prev) => (prev ? { ...prev, favorites: prev.favorites + delta } : prev));

    setAssets((prev) =>
      prev.map((a) => (a.id === assetId ? { ...a, is_favorite: nextFav } : a))
    );
    adjustFavorites(nextFav ? 1 : -1);

    try {
      await toggleAssetFavorite(assetId, nextFav);
    } catch (err) {
      console.error("Failed to toggle favorite:", err);
      // Revert on error
      setAssets((prev) =>
        prev.map((a) => (a.id === assetId ? { ...a, is_favorite: !nextFav } : a))
      );
      adjustFavorites(nextFav ? -1 : 1);
    }
  };

  // Drawer: open in place; reload counts on close if anything changed.
  const handleOpen = (asset: Asset) => {
    drawerChanged.current = false;
    setOpenAssetId(asset.id);
  };
  const closeDrawer = () => {
    setOpenAssetId(null);
    if (drawerChanged.current) {
      drawerChanged.current = false;
      setReloadToken((t) => t + 1);
    }
  };
  const handleDrawerChanged = (updated: Asset) => {
    drawerChanged.current = true;
    setAssets((prev) => prev.map((a) => (a.id === updated.id ? { ...a, ...updated } : a)));
  };
  const handleDrawerDeleted = (id: number) => {
    setAssets((prev) => prev.filter((a) => a.id !== id));
    drawerChanged.current = true;
    closeDrawer();
  };

  // Select mode
  const toggleSelected = (asset: Asset) =>
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(asset.id)) next.delete(asset.id);
      else next.add(asset.id);
      return next;
    });
  const exitSelectMode = () => {
    setSelectMode(false);
    setSelectedIds(new Set());
  };
  const handleBulkApply = async (change: Omit<BulkUpdateInput, "asset_ids">, label: string) => {
    setBulkBusy(true);
    try {
      const { updated } = await bulkUpdateAssets({ asset_ids: [...selectedIds], ...change });
      toast.success(t.library.bulkDone(label, updated));
      setReloadToken((t) => t + 1);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t.library.bulkFailed);
    } finally {
      setBulkBusy(false);
    }
  };

  const handleRetry = () => {
    setIsCategoriesLoading(true);
    setReloadToken((t) => t + 1);
  };

  // Computed properties
  const hasActiveFilters = Boolean(
    debouncedSearch.trim() ||
      selectedCategory !== "all" ||
      selectedTags.length > 0 ||
      isFavoriteOnly ||
      hasPreview ||
      hasBooth ||
      localStatus !== "all" ||
      compatibleWith
  );

  return (
    <div className="flex min-h-screen flex-col">
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
          favoriteCount={stats?.favorites}
          availableTags={availableTags}
          selectedTags={selectedTags}
          onToggleTag={handleToggleTag}
          onClearTags={handleClearTags}
          isLoading={isCategoriesLoading}
          totalAssetsCount={stats?.total}
          categoryCounts={stats?.by_category}
          draftCount={stats?.drafts}
          avatars={avatars}
        />

        {/* Center/Right Asset Browsing View */}
        <main className={`min-w-0 flex-1 p-4 sm:p-6 lg:p-8 ${selectMode ? "pb-28" : ""}`}>
          {/* Title */}
          <div className="mb-3">
            <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
              {isFavoriteOnly && <Heart className="size-5 fill-rose-500 text-rose-500" />}
              {isFavoriteOnly ? t.library.favorites : selectedCategory === "all" ? t.library.allAssets : categoryName(selectedCategory)}
            </h1>
            <div className="mt-1 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
              <span>
                {isLoading ? t.common.loading : t.common.assets(assets.length)}
                {debouncedSearch && !isLoading && t.library.matching(debouncedSearch)}
              </span>
              {compatibleWith && (
                <span className="inline-flex items-center gap-1.5 rounded-md border border-violet-300 dark:border-violet-800/70 bg-violet-100 dark:bg-violet-950/40 px-2 py-0.5 text-xs text-violet-800 dark:text-violet-200">
                  <UserRound className="size-3.5" /> {t.library.compatibleWith(compatibleWith.name)}
                  <button type="button" onClick={() => setCompatibleWith(null)} aria-label={t.library.showAll} className="hover:text-foreground">
                    <X className="size-3.5" />
                  </button>
                </span>
              )}
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
            view={view}
            onViewChange={setView}
            size={cardSize}
            onSizeChange={setCardSize}
            selectMode={selectMode}
            onToggleSelectMode={() => (selectMode ? exitSelectMode() : setSelectMode(true))}
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
            onOpen={handleOpen}
            view={view}
            size={cardSize}
            selectable={selectMode}
            selectedIds={selectedIds}
            onToggleSelect={toggleSelected}
            welcome={stats?.total === 0 ? <WelcomePanel draftCount={stats.drafts} /> : undefined}
          />
        </main>
      </div>

      {selectMode && (
        <BulkActionBar
          count={selectedIds.size}
          totalVisible={assets.length}
          categories={categories}
          avatars={avatars}
          busy={bulkBusy}
          onApply={handleBulkApply}
          onSelectAll={() => setSelectedIds(new Set(assets.map((a) => a.id)))}
          onClear={() => setSelectedIds(new Set())}
          onExit={exitSelectMode}
        />
      )}

      <Sheet open={openAssetId !== null} onOpenChange={(open) => !open && closeDrawer()}>
        <SheetContent side="right" className="overflow-y-auto p-5 data-[side=right]:w-full data-[side=right]:sm:max-w-2xl sm:p-6">
          <SheetHeader className="sr-only">
            <SheetTitle>{t.common.assetDetails}</SheetTitle>
          </SheetHeader>
          {openAssetId !== null && (
            <AssetDetail
              key={openAssetId}
              assetId={openAssetId}
              variant="drawer"
              onChanged={handleDrawerChanged}
              onDeleted={handleDrawerDeleted}
            />
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}

export default function Home() {
  const { t } = useI18n();
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
          {t.library.loadingLibrary}
        </div>
      }
    >
      <LibraryView />
    </Suspense>
  );
}
