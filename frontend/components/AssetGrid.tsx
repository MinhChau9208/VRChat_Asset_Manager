"use client";

import React from "react";
import { Asset } from "@/lib/api";
import { AssetCard } from "./AssetCard";

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
}) => {
  // 1. Error State
  if (error) {
    return (
      <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-rose-900/40 bg-rose-950/20 p-8 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-rose-900/40 border border-rose-700/50 text-rose-400 mb-3 text-xl">
          ⚠️
        </div>
        <h3 className="text-base font-semibold text-rose-200 mb-1">
          Unable to load assets
        </h3>
        <p className="text-xs text-rose-400/80 max-w-md mb-4 font-mono">
          {error}
        </p>
        <button
          onClick={onRetry}
          className="inline-flex items-center gap-2 rounded-lg bg-rose-600 hover:bg-rose-500 text-white text-xs font-semibold px-4 py-2 transition-colors cursor-pointer"
        >
          <svg
            className="w-3.5 h-3.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          Retry Connection
        </button>
      </div>
    );
  }

  // 2. Loading State (Skeleton Grid)
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4">
        {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
          <div
            key={i}
            className="flex flex-col rounded-xl border border-neutral-800 bg-neutral-900/50 overflow-hidden animate-pulse"
          >
            <div className="aspect-[16/10] w-full bg-neutral-800/60" />
            <div className="p-3.5 space-y-2.5">
              <div className="h-4 w-3/4 rounded bg-neutral-800" />
              <div className="h-3 w-1/2 rounded bg-neutral-800/60" />
              <div className="flex gap-1 pt-1">
                <div className="h-4 w-12 rounded bg-neutral-800/40" />
                <div className="h-4 w-16 rounded bg-neutral-800/40" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  // 3. Empty States
  if (assets.length === 0) {
    if (isFavoriteOnly) {
      return (
        <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-rose-900/30 bg-rose-950/10 p-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-rose-950/60 border border-rose-800/60 text-rose-400 mb-3 text-2xl">
            ♥
          </div>
          <h3 className="text-base font-semibold text-white mb-1">
            No favorite assets yet
          </h3>
          <p className="text-xs text-neutral-400 max-w-sm mb-4">
            Click the heart icon (♡) on any asset card to save it here for quick access.
          </p>
          <button
            onClick={onClearFilters}
            className="rounded-lg border border-neutral-700 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-3.5 py-1.5 transition-colors cursor-pointer"
          >
            Browse All Assets
          </button>
        </div>
      );
    }

    if (searchQuery.trim() !== "") {
      return (
        <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-neutral-800/80 bg-neutral-900/40 p-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-800 text-neutral-400 mb-3 text-xl">
            🔍
          </div>
          <h3 className="text-base font-semibold text-white mb-1">
            No assets match your search
          </h3>
          <p className="text-xs text-neutral-400 max-w-sm mb-4">
            No results found for &ldquo;<span className="text-cyan-400 font-mono">{searchQuery}</span>&rdquo;. Try searching by name, author, or tags.
          </p>
          <button
            onClick={onClearFilters}
            className="rounded-lg border border-neutral-700 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-3.5 py-1.5 transition-colors cursor-pointer"
          >
            Clear Search
          </button>
        </div>
      );
    }

    if (hasActiveFilters) {
      return (
        <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-neutral-800/80 bg-neutral-900/40 p-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-800 text-neutral-400 mb-3 text-xl">
            🎯
          </div>
          <h3 className="text-base font-semibold text-white mb-1">
            No matching assets
          </h3>
          <p className="text-xs text-neutral-400 max-w-sm mb-4">
            No assets match the current combination of active filters.
          </p>
          <button
            onClick={onClearFilters}
            className="rounded-lg border border-neutral-700 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-3.5 py-1.5 transition-colors cursor-pointer"
          >
            Reset Filters
          </button>
        </div>
      );
    }

    if (selectedCategory !== "all") {
      return (
        <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-neutral-800/80 bg-neutral-900/40 p-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-neutral-800 text-neutral-400 mb-3 text-xl">
            📂
          </div>
          <h3 className="text-base font-semibold text-white mb-1">
            No assets in {selectedCategory}
          </h3>
          <p className="text-xs text-neutral-400 max-w-sm mb-4">
            There are currently no items categorized under {selectedCategory}.
          </p>
          <button
            onClick={onClearFilters}
            className="rounded-lg border border-neutral-700 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium px-3.5 py-1.5 transition-colors cursor-pointer"
          >
            View All Assets
          </button>
        </div>
      );
    }

    // Completely empty library
    return (
      <div className="flex min-h-[360px] flex-col items-center justify-center rounded-2xl border border-dashed border-neutral-800 bg-neutral-900/20 p-8 text-center">
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 mb-3 text-2xl">
          ✨
        </div>
        <h3 className="text-base font-semibold text-white mb-1">
          No assets yet
        </h3>
        <p className="text-xs text-neutral-400 max-w-md mb-2">
          Your personal VRChat asset library is ready. Use the &ldquo;Add Asset&rdquo; button to add your first item.
        </p>
        <p className="text-[11px] text-neutral-500">
          Tip: You can populate sample assets using <code className="text-neutral-300 font-mono">go run ./cmd/seed</code> in backend.
        </p>
      </div>
    );
  }

  // 4. Asset Cards Grid
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4">
      {assets.map((asset) => (
        <AssetCard
          key={asset.id}
          asset={asset}
          onToggleFavorite={onToggleFavorite}
        />
      ))}
    </div>
  );
};
