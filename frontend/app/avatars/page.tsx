"use client";

import React, { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { ArrowLeft, ExternalLink, Info, Pencil, ShoppingBag, UserRound } from "lucide-react";
import { Asset, Category, buildCategoryTree, getAssetByID, getAssets, getCategories, toggleAssetFavorite } from "@/lib/api";
import { CategoryIcon } from "@/lib/categoryIcon";
import { AssetCard, assetPreviewSrc } from "@/components/AssetCard";
import { AssetDetail } from "@/components/AssetDetail";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useI18n } from "@/lib/i18n";

interface Group {
  name: string;
  assets: Asset[];
}

// Group key for assets without a category (translated when shown).
const UNCATEGORIZED = "";

/** Groups assets by top-level category, in the sidebar's order. */
function groupByCategory(assets: Asset[], categories: Category[]): Group[] {
  const rootOf = new Map<number, Category>();
  const tree = buildCategoryTree(categories);
  for (const root of tree) {
    rootOf.set(root.id, root);
    for (const child of root.children) rootOf.set(child.id, root);
  }
  const groups = new Map<string, Asset[]>();
  for (const a of assets) {
    const root = a.category_id !== null ? rootOf.get(a.category_id) : undefined;
    const key = root?.name ?? UNCATEGORIZED;
    groups.set(key, [...(groups.get(key) ?? []), a]);
  }
  const order = [...tree.map((r) => r.name), UNCATEGORIZED];
  return order.filter((name) => groups.has(name)).map((name) => ({ name, assets: groups.get(name)! }));
}

function AvatarRoute() {
  const id = Number(useSearchParams().get("id"));
  return <AvatarView key={id} id={id} />;
}

export default function AvatarPage() {
  return (
    <Suspense>
      <AvatarRoute />
    </Suspense>
  );
}

function AvatarView({ id }: { id: number }) {
  const [avatar, setAvatar] = useState<Asset | null>(null);
  const [items, setItems] = useState<Asset[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [openId, setOpenId] = useState<number | null>(null);
  const { t, categoryName } = useI18n();

  useEffect(() => {
    let cancelled = false;
    Promise.all([getAssetByID(id), getAssets({ compatible_with: id, sort: "name_asc" }), getCategories()])
      .then(([a, compatible, cats]) => {
        if (cancelled) return;
        setAvatar(a);
        setItems(compatible);
        setCategories(cats);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : t.avatar.loadFailed);
      });
    return () => {
      cancelled = true;
    };
  }, [id, reloadToken, t]);

  const toggleFavorite = async (assetId: number, next: boolean) => {
    setItems((prev) => prev.map((a) => (a.id === assetId ? { ...a, is_favorite: next } : a)));
    try {
      await toggleAssetFavorite(assetId, next);
    } catch {
      setItems((prev) => prev.map((a) => (a.id === assetId ? { ...a, is_favorite: !next } : a)));
    }
  };

  if (error) {
    return (
      <main className="flex min-h-screen flex-col items-center justify-center gap-3 p-6 text-center">
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button asChild variant="secondary" size="sm">
          <Link href="/">{t.common.backToLibrary}</Link>
        </Button>
      </main>
    );
  }

  const src = avatar ? assetPreviewSrc(avatar) : null;
  const groups = groupByCategory(items, categories);

  return (
    <main className="min-h-screen p-4 sm:p-6 md:p-8">
      <div className="mx-auto max-w-6xl space-y-8">
        <Link href="/" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary">
          <ArrowLeft className="size-4" /> {t.common.library}
        </Link>

        {/* Hero */}
        <section className="flex flex-col gap-6 rounded-2xl border border-border bg-gradient-to-br from-violet-100 dark:from-violet-950/40 via-card/40 to-card/20 p-6 sm:flex-row sm:items-center">
          <div className="size-36 shrink-0 overflow-hidden rounded-2xl border border-border bg-muted sm:size-44">
            {src ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={src} alt="" className="size-full object-cover" />
            ) : (
              <div className="flex size-full items-center justify-center">
                <UserRound className="size-12 text-muted-foreground/50" />
              </div>
            )}
          </div>
          <div className="min-w-0 flex-1 space-y-3">
            <div>
              <p className="text-xs font-medium uppercase tracking-wider text-violet-700 dark:text-violet-300/80">{t.avatar.label}</p>
              <h1 className="text-2xl font-semibold leading-tight sm:text-3xl">{avatar?.name ?? "…"}</h1>
              {avatar?.author && <p className="mt-1 text-sm text-muted-foreground">{t.common.by(avatar.author)}</p>}
            </div>
            <p className="text-sm text-muted-foreground">
              {avatar ? t.avatar.compatibleCount(items.length) : t.common.loading}
            </p>
            {avatar && (
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" onClick={() => setOpenId(avatar.id)}>
                  <Pencil /> {t.avatar.details}
                </Button>
                {avatar.booth_url && (
                  <Button asChild size="sm" className="bg-red-700 text-white hover:bg-red-600">
                    <a href={avatar.booth_url} target="_blank" rel="noopener noreferrer">
                      <ShoppingBag /> BOOTH <ExternalLink className="size-3.5 opacity-70" />
                    </a>
                  </Button>
                )}
              </div>
            )}
          </div>
        </section>

        {/* Compatible assets */}
        {avatar && items.length === 0 && (
          <div className="flex items-start gap-3 rounded-xl border border-dashed border-border p-5 text-sm text-muted-foreground">
            <Info className="mt-0.5 size-4 shrink-0" />
            <div className="space-y-1">
              <p className="text-foreground">{t.avatar.nothing}</p>
              <p>{t.avatar.howTo}</p>
            </div>
          </div>
        )}

        {groups.map((group) => (
          <section key={group.name || "uncategorized"}>
            <h2 className="mb-3 flex items-center gap-2 text-lg font-semibold">
              <CategoryIcon name={group.name} className="size-5 text-muted-foreground" />
              {group.name ? categoryName(group.name) : t.common.uncategorized}
              <span className="text-sm font-normal text-muted-foreground">{group.assets.length}</span>
            </h2>
            <div className="grid grid-cols-[repeat(auto-fill,minmax(170px,1fr))] gap-4">
              {group.assets.map((a) => (
                <AssetCard key={a.id} asset={a} onOpen={(x) => setOpenId(x.id)} onToggleFavorite={toggleFavorite} />
              ))}
            </div>
          </section>
        ))}
      </div>

      <Sheet
        open={openId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setOpenId(null);
            setReloadToken((t) => t + 1); // compatibility or categories may have changed
          }
        }}
      >
        <SheetContent side="right" className="overflow-y-auto p-5 data-[side=right]:w-full data-[side=right]:sm:max-w-2xl sm:p-6">
          <SheetHeader className="sr-only">
            <SheetTitle>{t.common.assetDetails}</SheetTitle>
          </SheetHeader>
          {openId !== null && (
            <AssetDetail
              key={openId}
              assetId={openId}
              variant="drawer"
              onDeleted={() => {
                setOpenId(null);
                setReloadToken((t) => t + 1);
              }}
            />
          )}
        </SheetContent>
      </Sheet>
    </main>
  );
}
