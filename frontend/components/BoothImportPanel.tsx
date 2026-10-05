"use client";

import React, { useState } from "react";
import { BoothSuggestion, CompatAvatar } from "@/lib/api";
import { UserRound, X } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export interface BoothSelection {
  name?: string;
  author?: string;
  categoryId?: number;
  tags: string[];
  compatibleAvatars: CompatAvatar[];
  /** Image to download as preview after saving, if any. */
  imageUrl?: string;
}

interface BoothImportPanelProps {
  suggestion: BoothSuggestion;
  onApply: (selection: BoothSelection) => void;
  onClose: () => void;
}

const checkClass = "accent-cyan-500 cursor-pointer";

/**
 * Shows what BOOTH says about an item and lets the user pick what to copy
 * into the form. Nothing is saved until the form itself is saved.
 */
export const BoothImportPanel: React.FC<BoothImportPanelProps> = ({ suggestion, onApply, onClose }) => {
  const { t, categoryName } = useI18n();
  const [useName, setUseName] = useState(true);
  const [useAuthor, setUseAuthor] = useState(Boolean(suggestion.author));
  const [useCategory, setUseCategory] = useState(suggestion.category_id !== null);
  const [tags, setTags] = useState<string[]>([]);
  const [compat, setCompat] = useState<CompatAvatar[]>(suggestion.compatible_avatars);
  const [image, setImage] = useState<string | null>(suggestion.images[0] ?? null);

  const toggleTag = (t: string) => setTags((prev) => (prev.includes(t) ? prev.filter((x) => x !== t) : [...prev, t]));
  const toggleCompat = (c: CompatAvatar) =>
    setCompat((prev) =>
      prev.some((x) => x.avatar_name === c.avatar_name) ? prev.filter((x) => x.avatar_name !== c.avatar_name) : [...prev, c]
    );

  const apply = () =>
    onApply({
      name: useName ? suggestion.name : undefined,
      author: useAuthor ? suggestion.author : undefined,
      categoryId: useCategory && suggestion.category_id !== null ? suggestion.category_id : undefined,
      tags,
      compatibleAvatars: compat,
      imageUrl: image ?? undefined,
    });

  return (
    <div className="mt-2 p-4 rounded-xl bg-background/90 border border-red-300 dark:border-red-900/50 space-y-4">
      <div className="flex items-start justify-between gap-3">
        <div className="text-xs text-muted-foreground">
          <span className="text-red-700 dark:text-red-300 font-semibold">BOOTH #{suggestion.item_id}</span> · {suggestion.booth_category}
          {suggestion.price && <> · {suggestion.price}</>}
          {suggestion.is_adult && <span className="ml-1 text-rose-600 dark:text-rose-400">· R18</span>}
        </div>
        <button type="button" onClick={onClose} className="text-muted-foreground hover:text-foreground text-xs cursor-pointer">
          <X className="size-4" />
        </button>
      </div>

      {/* Images */}
      {suggestion.images.length > 0 && (
        <div>
          <span className="block text-xs text-muted-foreground mb-1.5">{t.booth.previewImage}</span>
          <div className="flex gap-2 overflow-x-auto pb-1">
            <button
              type="button"
              onClick={() => setImage(null)}
              className={`h-20 w-20 shrink-0 rounded-lg border text-xs text-muted-foreground cursor-pointer ${
                image === null ? "border-primary/40 ring-1 ring-ring" : "border-border"
              }`}
            >
              {t.booth.keepCurrent}
            </button>
            {suggestion.images.map((url) => (
              <button
                key={url}
                type="button"
                onClick={() => setImage(url)}
                className={`h-20 w-20 shrink-0 rounded-lg overflow-hidden border cursor-pointer ${
                  image === url ? "border-primary/40 ring-1 ring-ring" : "border-border opacity-70 hover:opacity-100"
                }`}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={url} alt="" className="h-full w-full object-cover" referrerPolicy="no-referrer" />
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Fields */}
      <div className="space-y-1.5 text-xs">
        <label className="flex items-start gap-2 cursor-pointer">
          <input type="checkbox" checked={useName} onChange={(e) => setUseName(e.target.checked)} className={`${checkClass} mt-0.5`} />
          <span>
            <span className="text-muted-foreground">{t.booth.name}</span> <span className="text-foreground">{suggestion.name}</span>
          </span>
        </label>
        {suggestion.author && (
          <label className="flex items-start gap-2 cursor-pointer">
            <input type="checkbox" checked={useAuthor} onChange={(e) => setUseAuthor(e.target.checked)} className={`${checkClass} mt-0.5`} />
            <span>
              <span className="text-muted-foreground">{t.booth.author}</span> <span className="text-foreground">{suggestion.author}</span>
            </span>
          </label>
        )}
        {suggestion.category_id !== null && (
          <label className="flex items-start gap-2 cursor-pointer">
            <input type="checkbox" checked={useCategory} onChange={(e) => setUseCategory(e.target.checked)} className={`${checkClass} mt-0.5`} />
            <span>
              <span className="text-muted-foreground">{t.booth.category}</span> <span className="text-foreground">{categoryName(suggestion.category_name)}</span>
            </span>
          </label>
        )}
      </div>

      {suggestion.compatible_avatars.length > 0 && (
        <div>
          <span className="block text-xs text-muted-foreground mb-1.5">{t.booth.compatibleAvatars}</span>
          <div className="flex flex-wrap gap-1.5">
            {suggestion.compatible_avatars.map((c) => {
              const on = compat.some((x) => x.avatar_name === c.avatar_name);
              return (
                <button
                  key={c.avatar_name}
                  type="button"
                  onClick={() => toggleCompat(c)}
                  className={`px-2 py-0.5 rounded-md text-xs border cursor-pointer ${
                    on ? "bg-violet-100 dark:bg-violet-950/60 text-violet-800 dark:text-violet-200 border-violet-300 dark:border-violet-700" : "bg-card text-muted-foreground border-border line-through"
                  }`}
                  title={c.avatar_asset_id !== null ? t.common.inLibrary : t.common.notInLibrary}
                >
                  <UserRound className="inline size-3.5" /> {c.avatar_name}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {suggestion.tags.length > 0 && (
        <div>
          <span className="block text-xs text-muted-foreground mb-1.5">{t.booth.tags}</span>
          <div className="flex flex-wrap gap-1">
            {suggestion.tags.map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => toggleTag(t)}
                className={`px-2 py-0.5 rounded text-xs border cursor-pointer ${
                  tags.includes(t) ? "bg-primary/15 text-primary border-primary/40" : "bg-card text-muted-foreground border-border"
                }`}
              >
                #{t}
              </button>
            ))}
          </div>
        </div>
      )}

      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={onClose}
          className="px-3 py-1.5 rounded-lg text-xs text-muted-foreground hover:text-foreground bg-card border border-border cursor-pointer"
        >
          {t.common.cancel}
        </button>
        <button
          type="button"
          onClick={apply}
          className="px-3 py-1.5 rounded-lg text-xs font-semibold text-white bg-red-700 hover:bg-red-600 cursor-pointer"
        >
          {t.booth.useSelected}
        </button>
      </div>
    </div>
  );
};
