"use client";

import React from "react";
import { FolderOpen, Heart, RefreshCw, SearchX, SlidersHorizontal, Sparkles, TriangleAlert } from "lucide-react";
import { Asset } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { AssetCard, CardSize, ViewMode } from "./AssetCard";
import { cn } from "@/lib/utils";

interface AssetGridProps {
  assets: Asset[];
  isLoading: boolean;
  error: string | null;
  onRetry: () => void;
  searchQuery: string;
  selectedCategory: string;
  isFavoriteOnly?: boolean;
  hasActiveFilters?: boolean;
  onClearFilters: () => void;
  onToggleFavorite?: (assetId: number, nextFav: boolean) => void;
  onOpen?: (asset: Asset) => void;
  size?: CardSize;
  view?: ViewMode;
  selectable?: boolean;
  selectedIds?: Set<number>;
  onToggleSelect?: (asset: Asset) => void;
  /** Replaces the plain "No assets yet" state when the library is empty. */
  welcome?: React.ReactNode;
}

// Minimum card width per size; columns fill the available space.
const GRID_COLUMNS: Record<CardSize, string> = {
  sm: "grid-cols-[repeat(auto-fill,minmax(130px,1fr))] gap-3",
  md: "grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-4",
  lg: "grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-5",
};

function EmptyState({
  icon: Icon,
  title,
  text,
  action,
  onAction,
  tone = "neutral",
}: {
  icon: React.ElementType;
  title: string;
  text: React.ReactNode;
  action?: string;
  onAction?: () => void;
  tone?: "neutral" | "error";
}) {
  return (
    <div
      className={cn(
        "flex min-h-[360px] flex-col items-center justify-center rounded-2xl border p-8 text-center",
        tone === "error" ? "border-destructive/40 bg-destructive/5" : "border-dashed border-border bg-card/20"
      )}
    >
      <div
        className={cn(
          "mb-3 flex size-12 items-center justify-center rounded-full",
          tone === "error" ? "bg-destructive/15 text-destructive" : "bg-muted text-muted-foreground"
        )}
      >
        <Icon className="size-6" />
      </div>
      <h3 className="mb-1 text-base font-semibold text-foreground">{title}</h3>
      <p className="mb-4 max-w-md text-sm text-muted-foreground">{text}</p>
      {action && onAction && (
        <Button variant={tone === "error" ? "destructive" : "secondary"} size="sm" onClick={onAction}>
          {tone === "error" && <RefreshCw />}
          {action}
        </Button>
      )}
    </div>
  );
}

export const AssetGrid: React.FC<AssetGridProps> = ({
  assets,
  isLoading,
  error,
  onRetry,
  searchQuery,
  selectedCategory,
  isFavoriteOnly = false,
  hasActiveFilters = false,
  onClearFilters,
  onToggleFavorite,
  onOpen,
  size = "md",
  view = "grid",
  selectable = false,
  selectedIds,
  onToggleSelect,
  welcome,
}) => {
  if (error) {
    return (
      <EmptyState
        icon={TriangleAlert}
        tone="error"
        title="Unable to load assets"
        text={<span className="font-mono text-xs">{error}</span>}
        action="Retry"
        onAction={onRetry}
      />
    );
  }

  if (isLoading) {
    return view === "list" ? (
      <div className="space-y-1">
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} className="h-16 animate-pulse rounded-lg bg-muted/30" />
        ))}
      </div>
    ) : (
      <div className={cn("grid", GRID_COLUMNS[size])}>
        {Array.from({ length: 10 }, (_, i) => (
          <div key={i} className="overflow-hidden rounded-xl border border-border bg-card/40">
            <div className="aspect-square w-full animate-pulse bg-muted/40" />
            <div className="space-y-2 p-3">
              <div className="h-4 w-3/4 animate-pulse rounded bg-muted/60" />
              <div className="h-3 w-1/2 animate-pulse rounded bg-muted/40" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (assets.length === 0) {
    if (isFavoriteOnly) {
      return (
        <EmptyState
          icon={Heart}
          title="No favorite assets yet"
          text="Click the heart on any asset to keep it here."
          action="Browse all assets"
          onAction={onClearFilters}
        />
      );
    }
    if (searchQuery.trim() !== "") {
      return (
        <EmptyState
          icon={SearchX}
          title="No assets match your search"
          text={
            <>
              Nothing found for &ldquo;<span className="text-primary">{searchQuery}</span>&rdquo;. Try a name, author or tag.
            </>
          }
          action="Clear search"
          onAction={onClearFilters}
        />
      );
    }
    if (hasActiveFilters) {
      return (
        <EmptyState
          icon={SlidersHorizontal}
          title="No matching assets"
          text="No assets match the current filters."
          action="Reset filters"
          onAction={onClearFilters}
        />
      );
    }
    if (selectedCategory !== "all") {
      return (
        <EmptyState
          icon={FolderOpen}
          title={`No assets in ${selectedCategory}`}
          text="Nothing is in this category yet."
          action="View all assets"
          onAction={onClearFilters}
        />
      );
    }
    if (welcome) return <>{welcome}</>;
    return (
      <EmptyState
        icon={Sparkles}
        title="No assets yet"
        text="Add your first asset with “Add Asset”, or scan your asset folder from Scan & Review."
      />
    );
  }

  const cards = assets.map((asset) => (
    <AssetCard
      key={asset.id}
      asset={asset}
      size={size}
      view={view}
      onToggleFavorite={onToggleFavorite}
      onOpen={onOpen}
      selectable={selectable}
      selected={selectedIds?.has(asset.id)}
      onToggleSelect={onToggleSelect}
    />
  ));

  return view === "list" ? <div className="space-y-0.5">{cards}</div> : <div className={cn("grid", GRID_COLUMNS[size])}>{cards}</div>;
};
