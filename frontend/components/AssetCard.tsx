"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Check, Heart, TriangleAlert } from "lucide-react";
import { Asset, getAssetPreviewUrl } from "@/lib/api";
import { CategoryIcon } from "@/lib/categoryIcon";
import { cn } from "@/lib/utils";
import { assetHref } from "@/lib/routes";

export type CardSize = "sm" | "md" | "lg";
export type ViewMode = "grid" | "list";

interface AssetCardProps {
  asset: Asset;
  size?: CardSize;
  view?: ViewMode;
  onToggleFavorite?: (assetId: number, nextFav: boolean) => void;
  /** Opens the asset in place (drawer). Modifier-clicks still follow the link. */
  onOpen?: (asset: Asset) => void;
  selectable?: boolean;
  selected?: boolean;
  onToggleSelect?: (asset: Asset) => void;
}

export function assetPreviewSrc(asset: Asset): string | null {
  if (!asset.preview_path) return null;
  return /^https?:\/\//.test(asset.preview_path)
    ? asset.preview_path
    : getAssetPreviewUrl(asset.id, asset.updated_at);
}

function isModifiedClick(e: React.MouseEvent) {
  return e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0;
}

export const AssetCard: React.FC<AssetCardProps> = ({
  asset,
  size = "md",
  view = "grid",
  onToggleFavorite,
  onOpen,
  selectable = false,
  selected = false,
  onToggleSelect,
}) => {
  const [imageError, setImageError] = useState(false);
  const src = assetPreviewSrc(asset);
  const showImage = Boolean(src && !imageError);
  const categoryName = asset.category?.name;
  const missing = Boolean(asset.local_path) && asset.local_file_exists === false;

  const handleClick = (e: React.MouseEvent) => {
    if (isModifiedClick(e)) return;
    if (selectable && onToggleSelect) {
      e.preventDefault();
      onToggleSelect(asset);
    } else if (onOpen) {
      e.preventDefault();
      onOpen(asset);
    }
  };

  const favoriteButton = onToggleFavorite && (
    <button
      type="button"
      onClick={(e) => {
        e.preventDefault();
        e.stopPropagation();
        onToggleFavorite(asset.id, !asset.is_favorite);
      }}
      aria-label={asset.is_favorite ? "Remove from favorites" : "Add to favorites"}
      className={cn(
        "flex size-8 items-center justify-center rounded-full border backdrop-blur-sm transition-all",
        asset.is_favorite
          ? "border-rose-500/70 bg-rose-100 dark:bg-rose-950/80 text-rose-600 dark:text-rose-400"
          : "border-white/10 bg-black/50 text-white/80 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:text-rose-300"
      )}
    >
      <Heart className={cn("size-4", asset.is_favorite && "fill-current")} />
    </button>
  );

  const selectBox = selectable && (
    <span
      className={cn(
        "flex size-6 items-center justify-center rounded-md border-2 transition-colors",
        selected ? "border-primary bg-primary text-primary-foreground" : "border-white/60 bg-black/40"
      )}
    >
      {selected && <Check className="size-4" strokeWidth={3} />}
    </span>
  );

  const thumbnail = (className: string) => (
    <div className={cn("relative overflow-hidden bg-muted/40 flex items-center justify-center", className)}>
      {showImage && src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={src}
          alt={asset.name}
          onError={() => setImageError(true)}
          loading="lazy"
          className="size-full object-cover transition-transform duration-300 group-hover:scale-[1.04]"
        />
      ) : (
        <CategoryIcon name={categoryName} className="size-1/3 max-h-12 max-w-12 text-muted-foreground/40" strokeWidth={1.5} />
      )}
    </div>
  );

  // ---- List row ----
  if (view === "list") {
    return (
      <Link
        href={assetHref(asset.id)}
        onClick={handleClick}
        className={cn(
          "group flex items-center gap-3 rounded-lg border px-3 py-2 transition-colors",
          selected ? "border-primary/60 bg-primary/10" : "border-transparent hover:border-border hover:bg-card/60"
        )}
      >
        {selectBox}
        {thumbnail("size-12 shrink-0 rounded-md")}
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-foreground" title={asset.name}>
            {asset.name}
          </div>
          <div className="truncate text-xs text-muted-foreground">{asset.author || "Unknown author"}</div>
        </div>
        <div className="hidden w-40 items-center gap-1.5 text-xs text-muted-foreground md:flex">
          <CategoryIcon name={categoryName} className="size-3.5 shrink-0" />
          <span className="truncate">{asset.category?.name ?? "Uncategorized"}</span>
        </div>
        <div className="hidden w-48 gap-1 overflow-hidden lg:flex">
          {asset.tags.slice(0, 2).map((t) => (
            <span key={t} className="truncate rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              #{t}
            </span>
          ))}
        </div>
        {missing && (
          <span title="Files missing from disk">
            <TriangleAlert className="size-4 text-amber-600 dark:text-amber-400" />
          </span>
        )}
        <div className="w-8">{favoriteButton}</div>
      </Link>
    );
  }

  // ---- Grid card ----
  return (
    <Link
      href={assetHref(asset.id)}
      onClick={handleClick}
      className={cn(
        "group flex flex-col overflow-hidden rounded-xl border bg-card/50 transition-all duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        selected
          ? "border-primary ring-2 ring-primary/40"
          : "border-border hover:border-primary/40 hover:bg-card hover:shadow-lg hover:shadow-black/30"
      )}
    >
      <div className="relative">
        {thumbnail("aspect-square w-full")}
        <div className="absolute left-2 top-2">{selectBox}</div>
        <div className="absolute right-2 top-2">{favoriteButton}</div>
        {missing && (
          <span
            className="absolute bottom-2 left-2 inline-flex items-center gap-1 rounded-md bg-black/70 px-1.5 py-0.5 text-xs text-amber-300 backdrop-blur-sm"
            title="Files missing from disk"
          >
            <TriangleAlert className="size-3.5" /> Missing
          </span>
        )}
      </div>

      <div className={cn("flex flex-1 flex-col gap-1", size === "sm" ? "p-2" : "p-3")}>
        <h3
          className={cn(
            "font-medium leading-snug text-foreground group-hover:text-primary",
            size === "sm" ? "line-clamp-1 text-xs" : "line-clamp-2 text-sm"
          )}
          title={asset.name}
        >
          {asset.name}
        </h3>
        {size !== "sm" && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <CategoryIcon name={categoryName} className="size-3.5 shrink-0" />
            <span className="truncate">
              {asset.category?.name ?? "Uncategorized"}
              {asset.author && <> · {asset.author}</>}
            </span>
          </div>
        )}
        {size === "lg" && asset.tags.length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {asset.tags.slice(0, 4).map((t) => (
              <span key={t} className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                #{t}
              </span>
            ))}
          </div>
        )}
      </div>
    </Link>
  );
};
