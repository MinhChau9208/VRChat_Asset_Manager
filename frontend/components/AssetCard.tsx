"use client";

import React, { useState } from "react";
import Link from "next/link";
import { Asset, getAssetPreviewUrl } from "@/lib/api";

interface AssetCardProps {
  asset: Asset;
  onToggleFavorite?: (assetId: number, nextFav: boolean) => void;
}

export const AssetCard: React.FC<AssetCardProps> = ({ asset, onToggleFavorite }) => {
  const [imageError, setImageError] = useState(false);
  const [isFav, setIsFav] = useState(Boolean(asset.is_favorite));
  const [togglingFav, setTogglingFav] = useState(false);

  // Sync state if asset changes
  React.useEffect(() => {
    setIsFav(Boolean(asset.is_favorite));
  }, [asset.is_favorite]);

  const handleFavoriteClick = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (togglingFav) return;

    const nextVal = !isFav;
    setIsFav(nextVal);
    setTogglingFav(true);

    try {
      if (onToggleFavorite) {
        onToggleFavorite(asset.id, nextVal);
      } else {
        const { toggleAssetFavorite } = await import("@/lib/api");
        await toggleAssetFavorite(asset.id, nextVal);
      }
    } catch (err) {
      console.error("Failed to toggle favorite:", err);
      setIsFav(!nextVal); // Revert on failure
    } finally {
      setTogglingFav(false);
    }
  };

  const categoryName = asset.category?.name || "Asset";
  const previewSrc = asset.preview_path
    ? asset.preview_path.startsWith("http://") || asset.preview_path.startsWith("https://")
      ? asset.preview_path
      : getAssetPreviewUrl(asset.id, asset.updated_at)
    : null;
  const hasPreview = Boolean(previewSrc && !imageError);

  return (
    <Link
      href={`/assets/${asset.id}`}
      className="group flex flex-col rounded-xl border border-neutral-800/80 bg-neutral-900/60 overflow-hidden hover:border-cyan-500/50 hover:bg-neutral-900/90 hover:shadow-lg hover:shadow-cyan-950/20 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-cyan-500"
    >
      {/* Preview Area */}
      <div className="relative aspect-[16/10] w-full bg-neutral-950 overflow-hidden flex items-center justify-center border-b border-neutral-800/60">
        {hasPreview && previewSrc ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={previewSrc}
            alt={asset.name}
            onError={() => setImageError(true)}
            className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
            loading="lazy"
          />
        ) : (
          /* Polished Aesthetic Placeholder */
          <div className="flex flex-col items-center justify-center gap-1 text-neutral-600 group-hover:text-cyan-400/80 transition-colors">
            <div className="h-10 w-10 rounded-full bg-neutral-900 border border-neutral-800 flex items-center justify-center text-lg">
              📦
            </div>
            <span className="text-[10px] uppercase font-mono tracking-wider text-neutral-500">
              {categoryName}
            </span>
          </div>
        )}

        {/* Favorite Button on top-left */}
        <div className="absolute top-2 left-2 z-10">
          <button
            type="button"
            onClick={handleFavoriteClick}
            aria-label={isFav ? "Remove from favorites" : "Add to favorites"}
            title={isFav ? "Favorited" : "Add to favorites"}
            className={`flex items-center justify-center h-7 w-7 rounded-full text-xs font-bold border transition-all duration-200 ${
              isFav
                ? "bg-rose-950/80 border-rose-500/80 text-rose-400 shadow-sm shadow-rose-900/50 scale-105"
                : "bg-neutral-950/70 border-neutral-800 text-neutral-400 opacity-60 group-hover:opacity-100 hover:text-rose-400 hover:border-neutral-700 backdrop-blur-sm"
            }`}
          >
            {isFav ? "♥" : "♡"}
          </button>
        </div>

        {/* Category Badge overlay on top-right */}
        <div className="absolute top-2 right-2">
          <span className="inline-block px-2 py-0.5 rounded-full text-[10px] font-semibold bg-neutral-950/80 border border-neutral-800 text-neutral-300 backdrop-blur-sm">
            {categoryName}
          </span>
        </div>

        {/* Local file status badge on bottom-left of preview */}
        {asset.local_path && (
          <div className="absolute bottom-2 left-2">
            {asset.local_file_exists ? (
              <span
                className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[9px] font-mono font-medium bg-neutral-950/80 border border-emerald-800/60 text-emerald-400 backdrop-blur-sm"
                title="Local file exists on disk"
              >
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse" />
                Local
              </span>
            ) : (
              <span
                className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[9px] font-mono font-medium bg-neutral-950/80 border border-amber-800/60 text-amber-400 backdrop-blur-sm"
                title="Local path configured but missing on disk"
              >
                <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                Missing
              </span>
            )}
          </div>
        )}
      </div>

      {/* Card Content */}
      <div className="flex flex-1 flex-col p-3.5 justify-between gap-2.5">
        <div>
          {/* Asset Title */}
          <h3
            className="text-sm font-semibold text-white tracking-tight line-clamp-1 group-hover:text-cyan-300 transition-colors"
            title={asset.name}
          >
            {asset.name}
          </h3>

          {/* Author */}
          {asset.author ? (
            <p className="text-xs text-neutral-400 truncate mt-0.5">
              by <span className="text-neutral-300">@{asset.author}</span>
            </p>
          ) : (
            <p className="text-xs text-neutral-600 italic mt-0.5">
              Unknown author
            </p>
          )}
        </div>

        {/* Tags */}
        <div className="flex flex-wrap gap-1 items-center min-h-[22px]">
          {asset.tags && asset.tags.length > 0 ? (
            <>
              {asset.tags.slice(0, 3).map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono bg-neutral-800/80 text-neutral-400 border border-neutral-700/50"
                >
                  #{tag}
                </span>
              ))}
              {asset.tags.length > 3 && (
                <span className="text-[10px] font-mono text-neutral-500">
                  +{asset.tags.length - 3}
                </span>
              )}
            </>
          ) : (
            <span className="text-[10px] text-neutral-600 font-mono">
              no tags
            </span>
          )}
        </div>
      </div>
    </Link>
  );
};
