"use client";

import React, { useState } from "react";
import { Heart, Tag as TagIcon, UserRound, X } from "lucide-react";
import { Asset, BulkUpdateInput, Category, buildCategoryTree } from "@/lib/api";
import { Button } from "@/components/ui/button";

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
  const disabled = busy || count === 0;

  const addTag = () => {
    const t = tag.trim();
    if (!t) return;
    onApply({ add_tags: [t] }, `Added #${t}`);
    setTag("");
  };

  return (
    <div className="fixed inset-x-0 bottom-4 z-40 flex justify-center px-4">
      <div className="flex max-w-full flex-wrap items-center gap-2 rounded-xl border border-border bg-popover/95 px-3 py-2 shadow-2xl shadow-black/50 backdrop-blur">
        <span className="px-1 text-sm font-medium tabular-nums">{count} selected</span>
        <button
          type="button"
          onClick={count === totalVisible ? onClear : onSelectAll}
          className="text-xs text-primary hover:underline"
        >
          {count === totalVisible ? "Clear" : `Select all ${totalVisible}`}
        </button>
        <span className="mx-1 h-5 w-px bg-border" />

        <select
          value=""
          disabled={disabled}
          aria-label="Set category"
          onChange={(e) => {
            const v = e.target.value;
            if (!v) return;
            const id = v === "none" ? null : Number(v);
            const name = id === null ? "Uncategorized" : categories.find((c) => c.id === id)?.name;
            onApply({ set_category: true, category_id: id }, `Moved to ${name}`);
          }}
          className={selectClass}
        >
          <option value="">Set category…</option>
          {buildCategoryTree(categories).map((root) => (
            <React.Fragment key={root.id}>
              <option value={root.id}>{root.name}</option>
              {root.children.map((c) => (
                <option key={c.id} value={c.id}>
                  &nbsp;&nbsp;└ {c.name}
                </option>
              ))}
            </React.Fragment>
          ))}
          <option value="none">— Uncategorized</option>
        </select>

        <div className="flex items-center">
          <TagIcon className="pointer-events-none relative left-6 size-3.5 text-muted-foreground" />
          <input
            value={tag}
            disabled={disabled}
            onChange={(e) => setTag(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && addTag()}
            placeholder="Add tag…"
            className="h-8 w-32 rounded-md border border-input bg-transparent pl-7 pr-2 text-sm placeholder:text-muted-foreground focus:border-ring focus:outline-none"
          />
        </div>

        {avatars.length > 0 && (
          <div className="flex items-center">
            <UserRound className="pointer-events-none relative left-6 size-3.5 text-muted-foreground" />
            <select
              value=""
              disabled={disabled}
              aria-label="Mark compatible with avatar"
              onChange={(e) => {
                const avatar = avatars.find((a) => a.id === Number(e.target.value));
                if (avatar) {
                  onApply(
                    { add_compatible_avatars: [{ avatar_asset_id: avatar.id, avatar_name: avatar.name }] },
                    `Marked compatible with ${avatar.name}`
                  );
                }
              }}
              className={`${selectClass} pl-7`}
            >
              <option value="">Compatible with…</option>
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
          onClick={() => onApply({ is_favorite: true }, "Added to favorites")}
        >
          <Heart /> Favorite
        </Button>

        <Button variant="ghost" size="icon" onClick={onExit} aria-label="Exit select mode">
          <X />
        </Button>
      </div>
    </div>
  );
};
