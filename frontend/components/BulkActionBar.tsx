"use client";

import React, { useState } from "react";
import { Heart, Tag as TagIcon, UserRound, X } from "lucide-react";
import { Asset, BulkUpdateInput, Category, buildCategoryTree } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { useI18n } from "@/lib/i18n";

interface BulkActionBarProps {
  count: number;
  totalVisible: number;
  categories: Category[];
  avatars: Asset[];
  busy: boolean;
  onApply: (change: Omit<BulkUpdateInput, "asset_ids">, label: string) => void;
  onSelectAll: () => void;
  onClear: () => void;
  onExit: () => void;
}

const selectClass =
  "h-8 max-w-44 rounded-md border border-input bg-transparent px-2 text-sm text-foreground focus:border-ring focus:outline-none cursor-pointer [&>option]:bg-popover";

/**
 * Floating bar shown in select mode: apply a category, a tag, a compatible
 * avatar or favorite to every selected asset.
 */
export const BulkActionBar: React.FC<BulkActionBarProps> = ({
  count,
  totalVisible,
  categories,
  avatars,
  busy,
  onApply,
  onSelectAll,
  onClear,
  onExit,
}) => {
  const [tag, setTag] = useState("");
  const { t, categoryName } = useI18n();
  const disabled = busy || count === 0;

  const addTag = () => {
    const value = tag.trim();
    if (!value) return;
    onApply({ add_tags: [value] }, t.bulk.addedTag(value));
    setTag("");
  };

  return (
    <div className="fixed inset-x-0 bottom-4 z-40 flex justify-center px-4">
      <div className="flex max-w-full flex-wrap items-center gap-2 rounded-xl border border-border bg-popover/95 px-3 py-2 shadow-2xl shadow-black/50 backdrop-blur">
        <span className="px-1 text-sm font-medium tabular-nums">{t.bulk.selected(count)}</span>
        <button
          type="button"
          onClick={count === totalVisible ? onClear : onSelectAll}
          className="text-xs text-primary hover:underline"
        >
          {count === totalVisible ? t.bulk.clear : t.bulk.selectAll(totalVisible)}
        </button>
        <span className="mx-1 h-5 w-px bg-border" />

        <select
          value=""
          disabled={disabled}
          aria-label={t.bulk.setCategory}
          onChange={(e) => {
            const v = e.target.value;
            if (!v) return;
            const id = v === "none" ? null : Number(v);
            const cat = categories.find((c) => c.id === id);
            const name = cat ? categoryName(cat.name) : t.common.uncategorized;
            onApply({ set_category: true, category_id: id }, t.bulk.movedTo(name));
          }}
          className={selectClass}
        >
          <option value="">{t.bulk.setCategory}</option>
          {buildCategoryTree(categories).map((root) => (
            <React.Fragment key={root.id}>
              <option value={root.id}>{categoryName(root.name)}</option>
              {root.children.map((c) => (
                <option key={c.id} value={c.id}>
                  &nbsp;&nbsp;└ {categoryName(c.name)}
                </option>
              ))}
            </React.Fragment>
          ))}
          <option value="none">{t.bulk.noCategory}</option>
        </select>

        <div className="flex items-center">
          <TagIcon className="pointer-events-none relative left-6 size-3.5 text-muted-foreground" />
          <input
            value={tag}
            disabled={disabled}
            onChange={(e) => setTag(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addTag()}
            placeholder={t.bulk.addTag}
            className="h-8 w-32 rounded-md border border-input bg-transparent pl-7 pr-2 text-sm placeholder:text-muted-foreground focus:border-ring focus:outline-none"
          />
        </div>

        {avatars.length > 0 && (
          <div className="flex items-center">
            <UserRound className="pointer-events-none relative left-6 size-3.5 text-muted-foreground" />
            <select
              value=""
              disabled={disabled}
              aria-label={t.bulk.compatAria}
              onChange={(e) => {
                const avatar = avatars.find((a) => a.id === Number(e.target.value));
                if (avatar) {
                  onApply(
                    { add_compatible_avatars: [{ avatar_asset_id: avatar.id, avatar_name: avatar.name }] },
                    t.bulk.markedCompat(avatar.name)
                  );
                }
              }}
              className={`${selectClass} pl-7`}
            >
              <option value="">{t.bulk.compatWith}</option>
              {avatars.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </div>
        )}

        <Button
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => onApply({ is_favorite: true }, t.bulk.addedFavorites)}
        >
          <Heart /> {t.bulk.favorite}
        </Button>

        <Button variant="ghost" size="icon" onClick={onExit} aria-label={t.bulk.exit}>
          <X />
        </Button>
      </div>
    </div>
  );
};
