"use client";

import React from "react";
import Link from "next/link";
import { Heart, LayoutGrid, ScanSearch, Settings2 } from "lucide-react";
import { Asset, Category, buildCategoryTree } from "@/lib/api";
import { categoryIcon } from "@/lib/categoryIcon";
import { assetPreviewSrc } from "./AssetCard";
import { cn } from "@/lib/utils";
import { avatarHref } from "@/lib/routes";

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
  categoryCounts?: Record<string, number>;
  draftCount?: number;
  /** Avatars in the library, linked to their avatar pages. */
  avatars?: Asset[];
}

function NavItem({
  active,
  onClick,
  href,
  icon: Icon,
  label,
  count,
  indent = false,
  tone = "default",
}: {
  active?: boolean;
  onClick?: () => void;
  href?: string;
  icon?: React.ElementType;
  label: string;
  count?: number;
  indent?: boolean;
  tone?: "default" | "favorite" | "review";
}) {
  const className = cn(
    "flex w-full items-center gap-2.5 whitespace-nowrap rounded-md px-2.5 py-1.5 text-left text-sm transition-colors",
    indent && "md:pl-9",
    active
      ? tone === "favorite"
        ? "bg-rose-500/15 text-rose-700 dark:text-rose-300"
        : "bg-primary/15 text-primary"
      : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
  );
  const content = (
    <>
      {Icon && <Icon className={cn("size-4 shrink-0", tone === "favorite" && "text-rose-600 dark:text-rose-400")} />}
      <span className="flex-1 truncate">{label}</span>
      {count !== undefined && count > 0 && (
        <span
          className={cn(
            "text-xs tabular-nums",
            tone === "review" ? "rounded bg-amber-500/15 px-1.5 text-amber-700 dark:text-amber-300" : "text-muted-foreground/70"
          )}
        >
          {count}
        </span>
      )}
    </>
  );
  return href ? (
    <Link href={href} className={className}>
      {content}
    </Link>
  ) : (
    <button type="button" onClick={onClick} className={className}>
      {content}
    </button>
  );
}

function SectionTitle({ children, action }: { children: React.ReactNode; action?: React.ReactNode }) {
  return (
    <div className="mb-1.5 flex items-center justify-between px-2.5">
      <h2 className="text-xs font-medium uppercase tracking-wider text-muted-foreground/70">{children}</h2>
      {action}
    </div>
  );
}

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
  categoryCounts,
  draftCount,
  avatars = [],
}) => {
  const selectCategory = (name: string) => {
    if (isFavoriteOnly && onToggleFavoriteOnly) onToggleFavoriteOnly();
    onSelectCategory(name);
  };

  return (
    <aside className="flex w-full shrink-0 flex-col gap-5 [&>*]:shrink-0 border-b border-border p-3 md:sticky md:top-16 md:h-[calc(100vh-4rem)] md:w-60 md:overflow-y-auto md:border-b-0 md:border-r scrollbar-on-hover">
      <nav className="flex gap-0.5 overflow-x-auto md:flex-col">
        <NavItem
          icon={LayoutGrid}
          label="All Assets"
          count={totalAssetsCount}
          active={!isFavoriteOnly && selectedCategory === "all"}
          onClick={() => selectCategory("all")}
        />
        {onToggleFavoriteOnly && (
          <NavItem
            icon={Heart}
            label="Favorites"
            count={favoriteCount}
            tone="favorite"
            active={isFavoriteOnly}
            onClick={onToggleFavoriteOnly}
          />
        )}
        <NavItem icon={ScanSearch} label="Scan & Review" count={draftCount} tone="review" href="/review" />
      </nav>

      {avatars.length > 0 && (
        <div>
          <SectionTitle>Avatars</SectionTitle>
          <div className="flex gap-1 overflow-x-auto md:flex-col">
            {avatars.map((a) => {
              const src = assetPreviewSrc(a);
              return (
                <Link
                  key={a.id}
                  href={avatarHref(a.id)}
                  className="flex shrink-0 items-center gap-2.5 rounded-md px-2 py-1 text-sm text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
                >
                  <span className="size-7 shrink-0 overflow-hidden rounded-full border border-border bg-muted">
                    {src && (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={src} alt="" className="size-full object-cover" />
                    )}
                  </span>
                  <span className="truncate">{a.name}</span>
                </Link>
              );
            })}
          </div>
        </div>
      )}

      <div>
        <SectionTitle
          action={
            <Link href="/categories" className="text-muted-foreground/70 hover:text-primary" title="Manage categories">
              <Settings2 className="size-3.5" />
            </Link>
          }
        >
          Categories
        </SectionTitle>
        <nav className="flex gap-0.5 overflow-x-auto md:flex-col">
          {isLoading && categories.length === 0
            ? Array.from({ length: 8 }, (_, i) => <div key={i} className="h-8 w-full animate-pulse rounded-md bg-muted/40" />)
            : buildCategoryTree(categories).flatMap((root) => {
                const subtotal = root.children.reduce(
                  (sum, child) => sum + (categoryCounts?.[child.id] ?? 0),
                  categoryCounts?.[root.id] ?? 0
                );
                const isActive = (c: Category) =>
                  !isFavoriteOnly && selectedCategory.toLowerCase() === c.name.toLowerCase();
                return [
                  <NavItem
                    key={root.id}
                    icon={categoryIcon(root.name)}
                    label={root.name}
                    count={subtotal}
                    active={isActive(root)}
                    onClick={() => selectCategory(root.name)}
                  />,
                  ...root.children.map((child) => (
                    <NavItem
                      key={child.id}
                      label={child.name}
                      count={categoryCounts?.[child.id]}
                      indent
                      active={isActive(child)}
                      onClick={() => selectCategory(child.name)}
                    />
                  )),
                ];
              })}
        </nav>
      </div>

      {availableTags.length > 0 && onToggleTag && (
        <div>
          <SectionTitle
            action={
              selectedTags.length > 0 &&
              onClearTags && (
                <button type="button" onClick={onClearTags} className="text-xs text-primary hover:underline">
                  Clear ({selectedTags.length})
                </button>
              )
            }
          >
            Tags
          </SectionTitle>
          <div className="flex max-h-48 flex-wrap gap-1 overflow-y-auto px-1.5 md:max-h-none md:overflow-visible">
            {availableTags.map((tag) => {
              const on = selectedTags.includes(tag);
              return (
                <button
                  key={tag}
                  type="button"
                  onClick={() => onToggleTag(tag)}
                  className={cn(
                    "rounded-md border px-2 py-0.5 text-xs transition-colors",
                    on
                      ? "border-primary/50 bg-primary/15 text-primary"
                      : "border-border text-muted-foreground hover:border-muted-foreground/40 hover:text-foreground"
                  )}
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
