"use client";

import React from "react";

export type SortOption = "recent" | "updated" | "name_asc" | "name_desc";
export type LocalStatusOption = "all" | "available" | "missing" | "not_specified";

interface FilterToolbarProps {
  // Sort
  sort: SortOption;
  onSortChange: (sort: SortOption) => void;

  // Toggle Filters
  hasPreview: boolean;
  onToggleHasPreview: () => void;
  hasBooth: boolean;
  onToggleHasBooth: () => void;
  localStatus: LocalStatusOption;
  onLocalStatusChange: (status: LocalStatusOption) => void;

  // Search & Tags
  searchQuery: string;
  onClearSearch: () => void;
  selectedTags: string[];
  onRemoveTag: (tag: string) => void;

  // Favorite toggle state
  isFavoriteOnly: boolean;
  onToggleFavoriteOnly: () => void;

  // Overall counts & clear
  totalCount: number;
  hasActiveFilters: boolean;
  onClearAllFilters: () => void;
}

export const FilterToolbar: React.FC<FilterToolbarProps> = ({
  sort,
  onSortChange,
  hasPreview,
  onToggleHasPreview,
  hasBooth,
  onToggleHasBooth,
  localStatus,
  onLocalStatusChange,
  searchQuery,
  onClearSearch,
  selectedTags,
  onRemoveTag,
  isFavoriteOnly,
  onToggleFavoriteOnly,
  totalCount,
  hasActiveFilters,
  onClearAllFilters,
}) => {
  return (
    <div className="flex flex-col gap-3 py-3 border-b border-neutral-800/80 mb-4">
      {/* Top row: controls */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* Left side: Quick Toggles */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Favorites quick toggle */}
          <button
            type="button"
            onClick={onToggleFavoriteOnly}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all border ${
              isFavoriteOnly
                ? "bg-rose-950/60 text-rose-300 border-rose-600/80 shadow-sm shadow-rose-950/40"
                : "bg-neutral-900/80 text-neutral-400 border-neutral-800 hover:border-neutral-700 hover:text-neutral-200"
            }`}
          >
            <span className={isFavoriteOnly ? "text-rose-400" : "text-neutral-500"}>♥</span>
            Favorites
          </button>

          {/* Has Preview toggle */}
          <button
            type="button"
            onClick={onToggleHasPreview}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all border ${
              hasPreview
                ? "bg-cyan-950/60 text-cyan-300 border-cyan-600/80 shadow-sm shadow-cyan-950/40"
                : "bg-neutral-900/80 text-neutral-400 border-neutral-800 hover:border-neutral-700 hover:text-neutral-200"
            }`}
          >
            <span>🖼️</span>
            Has Preview
          </button>

          {/* Has BOOTH link toggle */}
          <button
            type="button"
            onClick={onToggleHasBooth}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all border ${
              hasBooth
                ? "bg-cyan-950/60 text-cyan-300 border-cyan-600/80 shadow-sm shadow-cyan-950/40"
                : "bg-neutral-900/80 text-neutral-400 border-neutral-800 hover:border-neutral-700 hover:text-neutral-200"
            }`}
          >
            <span>🛒</span>
            Has BOOTH
          </button>

          {/* Local file status dropdown */}
          <div className="flex items-center">
            <select
              value={localStatus}
              onChange={(e) => onLocalStatusChange(e.target.value as LocalStatusOption)}
              className="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-neutral-900 border border-neutral-800 text-neutral-300 hover:border-neutral-700 focus:border-cyan-500 focus:outline-none transition-colors cursor-pointer"
            >
              <option value="all">Local: All</option>
              <option value="available">Local: Available on disk</option>
              <option value="missing">Local: Missing on disk</option>
              <option value="not_specified">Local: Unconfigured</option>
            </select>
          </div>
        </div>

        {/* Right side: Sorting & Count */}
        <div className="flex items-center gap-3 ml-auto">
          <div className="text-xs text-neutral-400">
            <span className="font-semibold text-white font-mono">{totalCount}</span> {totalCount === 1 ? "asset" : "assets"}
          </div>

          <div className="flex items-center gap-1.5">
            <label htmlFor="sort-select" className="text-xs text-neutral-400">
              Sort:
            </label>
            <select
              id="sort-select"
              value={sort}
              onChange={(e) => onSortChange(e.target.value as SortOption)}
              className="px-2.5 py-1.5 rounded-lg text-xs font-medium bg-neutral-900 border border-neutral-800 text-neutral-200 hover:border-neutral-700 focus:border-cyan-500 focus:outline-none transition-colors cursor-pointer"
            >
              <option value="recent">Recently Added</option>
              <option value="updated">Recently Updated</option>
              <option value="name_asc">Name (A–Z)</option>
              <option value="name_desc">Name (Z–A)</option>
            </select>
          </div>
        </div>
      </div>

      {/* Bottom row: Active filter chips (if any) */}
      {hasActiveFilters && (
        <div className="flex flex-wrap items-center gap-1.5 pt-1 text-xs">
          <span className="text-neutral-500 text-[11px] uppercase tracking-wider font-mono mr-1">
            Active:
          </span>

          {/* Search Chip */}
          {searchQuery && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-800 text-cyan-300 border border-neutral-700 text-xs">
              <span>search: &ldquo;{searchQuery}&rdquo;</span>
              <button
                type="button"
                onClick={onClearSearch}
                className="text-neutral-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title="Remove search filter"
              >
                ×
              </button>
            </span>
          )}

          {/* Favorite Chip */}
          {isFavoriteOnly && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-rose-950/70 text-rose-300 border border-rose-800/80 text-xs">
              <span>Favorites</span>
              <button
                type="button"
                onClick={onToggleFavoriteOnly}
                className="text-rose-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title="Remove favorites filter"
              >
                ×
              </button>
            </span>
          )}

          {/* Preview Chip */}
          {hasPreview && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-cyan-950/70 text-cyan-300 border border-cyan-800/80 text-xs">
              <span>With Preview</span>
              <button
                type="button"
                onClick={onToggleHasPreview}
                className="text-cyan-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title="Remove preview filter"
              >
                ×
              </button>
            </span>
          )}

          {/* Booth Chip */}
          {hasBooth && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-cyan-950/70 text-cyan-300 border border-cyan-800/80 text-xs">
              <span>With BOOTH</span>
              <button
                type="button"
                onClick={onToggleHasBooth}
                className="text-cyan-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title="Remove booth filter"
              >
                ×
              </button>
            </span>
          )}

          {/* Local Status Chip */}
          {localStatus !== "all" && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-neutral-800 text-neutral-200 border border-neutral-700 text-xs">
              <span>
                Local: {localStatus === "available" ? "Available" : localStatus === "missing" ? "Missing" : "Unconfigured"}
              </span>
              <button
                type="button"
                onClick={() => onLocalStatusChange("all")}
                className="text-neutral-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title="Reset local status filter"
              >
                ×
              </button>
            </span>
          )}

          {/* Tag Chips */}
          {selectedTags.map((tag) => (
            <span
              key={tag}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md bg-cyan-950/70 text-cyan-300 border border-cyan-800/80 text-xs font-mono"
            >
              <span>#{tag}</span>
              <button
                type="button"
                onClick={() => onRemoveTag(tag)}
                className="text-cyan-400 hover:text-white ml-0.5 cursor-pointer font-bold"
                title={`Remove #${tag} filter`}
              >
                ×
              </button>
            </span>
          ))}

          {/* Clear All Button */}
          <button
            type="button"
            onClick={onClearAllFilters}
            className="ml-2 text-xs text-neutral-400 hover:text-cyan-400 underline underline-offset-2 transition-colors cursor-pointer"
          >
            Clear all
          </button>
        </div>
      )}
    </div>
  );
};
