"use client";

import React from "react";
import { Category } from "@/lib/api";

interface SidebarProps {
  categories: Category[];
  selectedCategory: string;
  onSelectCategory: (categoryName: string) => void;
  isFavoriteOnly?: boolean;
  onToggleFavoriteOnly?: () => void;
  favoriteCount?: number;
  availableTags?: string[];
  selectedTags?: string[];
  onToggleTag?: (tagName: string) => void;
  onClearTags?: () => void;
  isLoading: boolean;
  totalAssetsCount?: number;
}

// Category icons for visual polish
const getCategoryIcon = (name: string) => {
  switch (name.toLowerCase()) {
    case "avatar":
      return "👤";
    case "hair":
      return "💇";
    case "clothes":
      return "👗";
    case "shoes":
      return "👟";
    case "accessory":
      return "💍";
    case "gimmick":
      return "✨";
    case "texture":
      return "🎨";
    case "material":
      return "🔮";
    case "shader":
      return "🌈";
    case "other":
      return "📦";
    default:
      return "📁";
  }
};

export const Sidebar: React.FC<SidebarProps> = ({
  categories,
  selectedCategory,
  onSelectCategory,
  isFavoriteOnly = false,
  onToggleFavoriteOnly,
  favoriteCount,
  availableTags = [],
  selectedTags = [],
  onToggleTag,
  onClearTags,
  isLoading,
  totalAssetsCount,
}) => {
  return (
    <aside className="w-full md:w-56 shrink-0 md:min-h-[calc(100vh-4rem)] border-b md:border-b-0 md:border-r border-neutral-800 bg-neutral-950/40 p-4 flex flex-col gap-6">
      {/* Navigation / Library Section */}
      <div>
        <div className="mb-2.5 px-2 flex items-center justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
            Library
          </h2>
          {totalAssetsCount !== undefined && (
            <span className="text-[11px] font-mono text-neutral-400 bg-neutral-900 px-1.5 py-0.5 rounded border border-neutral-800">
              {totalAssetsCount}
            </span>
          )}
        </div>

        <nav className="flex md:flex-col gap-1 overflow-x-auto md:overflow-x-visible pb-2 md:pb-0 scrollbar-none">
          {/* All Assets */}
          <button
            type="button"
            onClick={() => {
              if (isFavoriteOnly && onToggleFavoriteOnly) {
                onToggleFavoriteOnly();
              }
              onSelectCategory("all");
            }}
            className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-xs font-medium transition-all whitespace-nowrap cursor-pointer text-left ${
              !isFavoriteOnly && selectedCategory === "all"
                ? "bg-cyan-500/15 text-cyan-400 border border-cyan-500/30 shadow-[0_0_12px_rgba(6,182,212,0.1)]"
                : "text-neutral-400 hover:bg-neutral-900/80 hover:text-neutral-200 border border-transparent"
            }`}
          >
            <span className="text-sm">🗂️</span>
            <span className="flex-1">All Assets</span>
          </button>

          {/* Favorites Filter */}
          {onToggleFavoriteOnly && (
            <button
              type="button"
              onClick={onToggleFavoriteOnly}
              className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-xs font-medium transition-all whitespace-nowrap cursor-pointer text-left ${
                isFavoriteOnly
                  ? "bg-rose-500/15 text-rose-400 border border-rose-500/30 shadow-[0_0_12px_rgba(244,63,94,0.1)] font-semibold"
                  : "text-neutral-400 hover:bg-neutral-900/80 hover:text-rose-300 border border-transparent"
              }`}
            >
              <span className="text-sm text-rose-500">♥</span>
              <span className="flex-1">Favorites</span>
              {favoriteCount !== undefined && favoriteCount > 0 && (
                <span className="text-[10px] font-mono text-rose-400 bg-rose-950/60 px-1.5 py-0.2 rounded border border-rose-800/40">
                  {favoriteCount}
                </span>
              )}
            </button>
          )}
        </nav>
      </div>

      {/* Categories Section */}
      <div>
        <div className="mb-2 px-2 flex items-center justify-between">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
            Categories
          </h2>
        </div>

        <nav className="flex md:flex-col gap-1 overflow-x-auto md:overflow-x-visible pb-2 md:pb-0 scrollbar-none">
          {/* Dynamic Categories from API */}
          {isLoading && categories.length === 0 ? (
            <div className="space-y-1.5 pt-1">
              {[1, 2, 3, 4, 5, 6].map((i) => (
                <div
                  key={i}
                  className="h-8 w-full rounded-lg bg-neutral-900/60 animate-pulse"
                />
              ))}
            </div>
          ) : (
            categories.map((cat) => {
              const isSelected = !isFavoriteOnly && selectedCategory.toLowerCase() === cat.name.toLowerCase();
              return (
                <button
                  key={cat.id}
                  type="button"
                  onClick={() => {
                    if (isFavoriteOnly && onToggleFavoriteOnly) {
                      onToggleFavoriteOnly();
                    }
                    onSelectCategory(cat.name);
                  }}
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-xs font-medium transition-all whitespace-nowrap cursor-pointer text-left ${
                    isSelected
                      ? "bg-cyan-500/15 text-cyan-400 border border-cyan-500/30 shadow-[0_0_12px_rgba(6,182,212,0.1)]"
                      : "text-neutral-400 hover:bg-neutral-900/80 hover:text-neutral-200 border border-transparent"
                  }`}
                >
                  <span className="text-sm">{getCategoryIcon(cat.name)}</span>
                  <span className="flex-1">{cat.name}</span>
                </button>
              );
            })
          )}
        </nav>
      </div>

      {/* Tags Filter Section */}
      {availableTags.length > 0 && onToggleTag && (
        <div>
          <div className="mb-2.5 px-2 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-neutral-400">
              Filter by Tag
            </h2>
            {selectedTags.length > 0 && onClearTags && (
              <button
                type="button"
                onClick={onClearTags}
                className="text-[10px] text-cyan-400 hover:text-cyan-300 transition-colors"
              >
                Clear ({selectedTags.length})
              </button>
            )}
          </div>

          <div className="flex flex-wrap gap-1.5 px-1 max-h-48 overflow-y-auto scrollbar-thin">
            {availableTags.map((tag) => {
              const isSelected = selectedTags.includes(tag);
              return (
                <button
                  key={tag}
                  type="button"
                  onClick={() => onToggleTag(tag)}
                  className={`inline-flex items-center px-2 py-1 rounded-md text-[11px] font-mono transition-all border ${
                    isSelected
                      ? "bg-cyan-500/20 text-cyan-300 border-cyan-500/50 shadow-sm"
                      : "bg-neutral-900/80 text-neutral-400 border-neutral-800 hover:border-neutral-700 hover:text-neutral-200"
                  }`}
                >
                  #{tag}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </aside>
  );
};
