"use client";

import React from "react";
import { CheckSquare, Heart, Image as ImageIcon, LayoutGrid, List, ShoppingBag, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { CardSize, ViewMode } from "./AssetCard";
import { cn } from "@/lib/utils";
import { useI18n } from "@/lib/i18n";

export type SortOption = "recent" | "updated" | "name_asc" | "name_desc";
export type LocalStatusOption = "all" | "available" | "missing" | "not_specified";

interface FilterToolbarProps {
  sort: SortOption;
  onSortChange: (sort: SortOption) => void;

  hasPreview: boolean;
  onToggleHasPreview: () => void;
  hasBooth: boolean;
  onToggleHasBooth: () => void;
  localStatus: LocalStatusOption;
  onLocalStatusChange: (status: LocalStatusOption) => void;

  searchQuery: string;
  onClearSearch: () => void;
  selectedTags: string[];
  onRemoveTag: (tag: string) => void;

  isFavoriteOnly: boolean;
  onToggleFavoriteOnly: () => void;

  totalCount: number;
  hasActiveFilters: boolean;
  onClearAllFilters: () => void;

  view: ViewMode;
  onViewChange: (view: ViewMode) => void;
  size: CardSize;
  onSizeChange: (size: CardSize) => void;
  selectMode: boolean;
  onToggleSelectMode: () => void;
}

const selectClass =
  "h-8 rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus:border-ring focus:outline-none cursor-pointer [&>option]:bg-popover";

function Chip({ active, onClick, icon: Icon, children }: { active: boolean; onClick: () => void; icon: React.ElementType; children: React.ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "inline-flex h-8 items-center gap-1.5 rounded-md border px-2.5 text-sm transition-colors",
        active
          ? "border-primary/50 bg-primary/15 text-primary"
          : "border-input text-muted-foreground hover:bg-muted/50 hover:text-foreground"
      )}
    >
      <Icon className="size-4" />
      {children}
    </button>
  );
}

function ActiveChip({ children, onRemove }: { children: React.ReactNode; onRemove: () => void }) {
  const { t } = useI18n();
  return (
    <span className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs text-foreground">
      {children}
      <button type="button" onClick={onRemove} className="text-muted-foreground hover:text-foreground" aria-label={t.toolbar.removeFilter}>
        <X className="size-3.5" />
      </button>
    </span>
  );
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
  hasActiveFilters,
  onClearAllFilters,
  view,
  onViewChange,
  size,
  onSizeChange,
  selectMode,
  onToggleSelectMode,
}) => {
  const { t } = useI18n();
  return (
    <div className="mb-4 flex flex-col gap-2.5 border-b border-border pb-3">
      <div className="flex flex-wrap items-center gap-2">
        <Chip active={isFavoriteOnly} onClick={onToggleFavoriteOnly} icon={Heart}>
          {t.toolbar.favorites}
        </Chip>
        <Chip active={hasPreview} onClick={onToggleHasPreview} icon={ImageIcon}>
          {t.toolbar.hasPreview}
        </Chip>
        <Chip active={hasBooth} onClick={onToggleHasBooth} icon={ShoppingBag}>
          {t.toolbar.hasBooth}
        </Chip>
        <select
          value={localStatus}
          onChange={(e) => onLocalStatusChange(e.target.value as LocalStatusOption)}
          className={selectClass}
          aria-label={t.toolbar.localFiles}
        >
          <option value="all">{t.toolbar.localAll}</option>
          <option value="available">{t.toolbar.onDisk}</option>
          <option value="missing">{t.toolbar.missingFromDisk}</option>
          <option value="not_specified">{t.toolbar.noPath}</option>
        </select>

        <div className="ml-auto flex items-center gap-2">
          <select value={sort} onChange={(e) => onSortChange(e.target.value as SortOption)} className={selectClass} aria-label={t.toolbar.sort}>
            <option value="recent">{t.toolbar.recent}</option>
            <option value="updated">{t.toolbar.updated}</option>
            <option value="name_asc">{t.toolbar.nameAsc}</option>
            <option value="name_desc">{t.toolbar.nameDesc}</option>
          </select>

          <ToggleGroup
            type="single"
            variant="outline"
            size="sm"
            value={view}
            onValueChange={(v) => v && onViewChange(v as ViewMode)}
            aria-label={t.toolbar.view}
          >
            <ToggleGroupItem value="grid" aria-label={t.toolbar.gridView}>
              <LayoutGrid />
            </ToggleGroupItem>
            <ToggleGroupItem value="list" aria-label={t.toolbar.listView}>
              <List />
            </ToggleGroupItem>
          </ToggleGroup>

          {view === "grid" && (
            <ToggleGroup
              type="single"
              variant="outline"
              size="sm"
              value={size}
              onValueChange={(v) => v && onSizeChange(v as CardSize)}
              aria-label={t.toolbar.cardSize}
            >
              <ToggleGroupItem value="sm" className="px-2.5 text-xs">S</ToggleGroupItem>
              <ToggleGroupItem value="md" className="px-2.5 text-xs">M</ToggleGroupItem>
              <ToggleGroupItem value="lg" className="px-2.5 text-xs">L</ToggleGroupItem>
            </ToggleGroup>
          )}

          <Button variant={selectMode ? "default" : "outline"} size="sm" onClick={onToggleSelectMode}>
            <CheckSquare />
            {t.toolbar.select}
          </Button>
        </div>
      </div>

      {hasActiveFilters && (
        <div className="flex flex-wrap items-center gap-1.5">
          {searchQuery && <ActiveChip onRemove={onClearSearch}>{t.toolbar.chipSearch(searchQuery)}</ActiveChip>}
          {isFavoriteOnly && <ActiveChip onRemove={onToggleFavoriteOnly}>{t.toolbar.chipFavorites}</ActiveChip>}
          {hasPreview && <ActiveChip onRemove={onToggleHasPreview}>{t.toolbar.chipHasPreview}</ActiveChip>}
          {hasBooth && <ActiveChip onRemove={onToggleHasBooth}>{t.toolbar.chipHasBooth}</ActiveChip>}
          {localStatus !== "all" && (
            <ActiveChip onRemove={() => onLocalStatusChange("all")}>
              {localStatus === "available" ? t.toolbar.chipOnDisk : localStatus === "missing" ? t.toolbar.chipMissing : t.toolbar.chipNoPath}
            </ActiveChip>
          )}
          {selectedTags.map((tag) => (
            <ActiveChip key={tag} onRemove={() => onRemoveTag(tag)}>
              #{tag}
            </ActiveChip>
          ))}
          <button type="button" onClick={onClearAllFilters} className="ml-1 text-xs text-muted-foreground hover:text-primary">
            {t.toolbar.clearAll}
          </button>
        </div>
      )}
    </div>
  );
};
