"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { toast } from "sonner";
import {
  CalendarDays,
  Check,
  ExternalLink,
  FileWarning,
  FolderOpen,
  Heart,
  Maximize2,
  MoreHorizontal,
  Pencil,
  ShoppingBag,
  Trash2,
  UserRound,
} from "lucide-react";
import {
  Asset,
  acceptDrafts,
  deleteAsset,
  deleteAssetPreview,
  getAssetByID,
  openAssetFolder,
  toggleAssetFavorite,
  uploadAssetPreview,
} from "@/lib/api";
import { CategoryIcon } from "@/lib/categoryIcon";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { AssetForm } from "./AssetForm";
import { AssetFilesPanel } from "./AssetFilesPanel";
import { PreviewDropzone } from "./PreviewDropzone";
import { assetPreviewSrc } from "./AssetCard";
import { cn } from "@/lib/utils";
import { assetHref, avatarHref } from "@/lib/routes";
import { useI18n } from "@/lib/i18n";

interface AssetDetailProps {
  assetId: number | string;
  variant: "page" | "drawer";
  /** Called whenever the asset changes (edit, favorite, preview, files). */
  onChanged?: (asset: Asset) => void;
  /** Called after the asset was deleted. */
  onDeleted?: (assetId: number) => void;
}

const errorMessage = (err: unknown, fallback: string) => (err instanceof Error ? err.message : fallback);

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground/70">{title}</h2>
      {children}
    </section>
  );
}

export function AssetDetail({ assetId, variant, onChanged, onDeleted }: AssetDetailProps) {
  const [asset, setAsset] = useState<Asset | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [isEditing, setIsEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [busy, setBusy] = useState<null | "preview" | "folder" | "delete">(null);
  const { t, categoryName, formatDate } = useI18n();

  useEffect(() => {
    let cancelled = false;
    getAssetByID(assetId)
      .then((a) => {
        if (!cancelled) setAsset(a);
      })
      .catch((err: unknown) => {
        if (!cancelled) setLoadError(errorMessage(err, t.detail.loadFailed));
      });
    return () => {
      cancelled = true;
    };
  }, [assetId, reloadToken, t]);

  const update = (next: Asset) => {
    setAsset(next);
    onChanged?.(next);
  };

  if (loadError) {
    return (
      <div className="flex flex-col items-center gap-3 py-16 text-center">
        <FileWarning className="size-8 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">{loadError}</p>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            setLoadError(null);
            setReloadToken((t) => t + 1);
          }}
        >
          {t.common.retry}
        </Button>
      </div>
    );
  }

  if (!asset) {
    return (
      <div className="grid gap-6 sm:grid-cols-[minmax(0,280px)_1fr]">
        <div className="aspect-square animate-pulse rounded-xl bg-muted/40" />
        <div className="space-y-3">
          <div className="h-7 w-3/4 animate-pulse rounded bg-muted/50" />
          <div className="h-4 w-1/3 animate-pulse rounded bg-muted/40" />
          <div className="h-24 animate-pulse rounded bg-muted/30" />
        </div>
      </div>
    );
  }

  const isAvatar = asset.category?.name.toLowerCase() === "avatar";
  const hasPath = Boolean(asset.local_path?.trim());
  const missing = hasPath && asset.local_file_exists === false;

  const handlePreviewFile = async (file: File) => {
    setBusy("preview");
    try {
      update(await uploadAssetPreview(asset.id, file));
      toast.success(t.detail.previewUpdated);
    } catch (err) {
      toast.error(errorMessage(err, t.detail.uploadFailed));
    } finally {
      setBusy(null);
    }
  };

  const handleRemovePreview = async () => {
    setBusy("preview");
    try {
      await deleteAssetPreview(asset.id);
      update({ ...asset, preview_path: "", updated_at: new Date().toISOString() });
      toast.success(t.detail.previewRemoved);
    } catch (err) {
      toast.error(errorMessage(err, t.detail.removeFailed));
    } finally {
      setBusy(null);
    }
  };

  const handleOpenFolder = async () => {
    setBusy("folder");
    try {
      await openAssetFolder(asset.id);
    } catch (err) {
      toast.error(errorMessage(err, t.detail.openFailed));
    } finally {
      setBusy(null);
    }
  };

  const handleFavorite = async () => {
    const next = !asset.is_favorite;
    update({ ...asset, is_favorite: next });
    try {
      await toggleAssetFavorite(asset.id, next);
    } catch {
      update({ ...asset, is_favorite: !next });
      toast.error(t.detail.favoriteFailed);
    }
  };

  const handleAccept = async () => {
    try {
      await acceptDrafts([asset.id]);
      update({ ...asset, status: "active" });
      toast.success(t.detail.addedToLibrary);
    } catch (err) {
      toast.error(errorMessage(err, t.detail.acceptFailed));
    }
  };

  const handleDelete = async () => {
    setBusy("delete");
    try {
      await deleteAsset(asset.id);
      setConfirmDelete(false);
      toast.success(t.detail.deleted(asset.name));
      onDeleted?.(asset.id);
    } catch (err) {
      toast.error(errorMessage(err, t.detail.deleteFailed));
      setBusy(null);
    }
  };

  if (isEditing) {
    return (
      <AssetForm
        mode="edit"
        initialData={asset}
        onSubmitSuccess={(updated) => {
          update(updated);
          setIsEditing(false);
          toast.success(t.detail.saved);
        }}
        onCancel={() => setIsEditing(false)}
      />
    );
  }

  return (
    <div className="space-y-5">
      {asset.status === "draft" && (
        <div className="flex flex-wrap items-center gap-3 rounded-lg border border-amber-300 dark:border-amber-700/50 bg-amber-100 dark:bg-amber-950/30 px-3 py-2 text-sm text-amber-800 dark:text-amber-200">
          <span className="flex-1">{t.detail.draftBanner}</span>
          <Button size="sm" onClick={handleAccept} className="bg-emerald-600 text-white hover:bg-emerald-500">
            <Check /> {t.common.accept}
          </Button>
        </div>
      )}

      {/* Title row (in the drawer, leave room for the sheet's close button) */}
      <div className={cn("flex items-start gap-3", variant === "drawer" && "pr-9")}>
        <div className="min-w-0 flex-1">
          <div className="mb-1.5 flex flex-wrap items-center gap-1.5">
            <Badge variant="secondary" className="gap-1">
              <CategoryIcon name={asset.category?.name} className="size-3.5" />
              {asset.category ? categoryName(asset.category.name) : t.common.uncategorized}
            </Badge>
          </div>
          <h1 className={cn("font-semibold leading-tight text-foreground", variant === "page" ? "text-2xl" : "text-xl")}>
            {asset.name}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">{asset.author ? t.common.by(asset.author) : t.common.unknownAuthor}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleFavorite}
            aria-label={asset.is_favorite ? t.card.removeFavorite : t.card.addFavorite}
            className={cn(asset.is_favorite && "text-rose-600 dark:text-rose-400 hover:text-rose-700 dark:hover:text-rose-300")}
          >
            <Heart className={cn(asset.is_favorite && "fill-current")} />
          </Button>
          <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
            <Pencil /> {t.common.edit}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" aria-label={t.detail.moreActions}>
                <MoreHorizontal />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {variant === "drawer" && (
                <DropdownMenuItem asChild>
                  <Link href={assetHref(asset.id)}>
                    <Maximize2 /> {t.detail.openFullPage}
                  </Link>
                </DropdownMenuItem>
              )}
              {isAvatar && (
                <DropdownMenuItem asChild>
                  <Link href={avatarHref(asset.id)}>
                    <UserRound /> {t.detail.avatarPage}
                  </Link>
                </DropdownMenuItem>
              )}
              <DropdownMenuSeparator />
              <DropdownMenuItem variant="destructive" onSelect={() => setConfirmDelete(true)}>
                <Trash2 /> {t.common.delete}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className={cn("grid gap-6", variant === "page" ? "md:grid-cols-[minmax(0,340px)_1fr]" : "sm:grid-cols-[minmax(0,240px)_1fr]")}>
        {/* Preview */}
        <div className="space-y-2">
          <PreviewDropzone
            src={assetPreviewSrc(asset)}
            onFile={handlePreviewFile}
            onError={(m) => toast.error(m)}
            busy={busy === "preview"}
            pasteAnywhere
          />
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span className="inline-flex items-center gap-1">
              <CalendarDays className="size-3.5" />
              {t.detail.added(formatDate(asset.created_at))}
            </span>
            {asset.preview_path && (
              <button type="button" onClick={handleRemovePreview} className="hover:text-destructive" disabled={busy !== null}>
                {t.detail.removePreview}
              </button>
            )}
          </div>
        </div>

        {/* Details */}
        <div className="min-w-0 space-y-5">
          <div className="flex flex-wrap gap-2">
            <Button onClick={handleOpenFolder} disabled={!hasPath || missing || busy === "folder"} variant="secondary">
              <FolderOpen /> {busy === "folder" ? t.detail.opening : t.detail.openFolder}
            </Button>
            {asset.booth_url && (
              <Button asChild className="bg-red-700 text-white hover:bg-red-600">
                <a href={asset.booth_url} target="_blank" rel="noopener noreferrer">
                  <ShoppingBag /> BOOTH <ExternalLink className="size-3.5 opacity-70" />
                </a>
              </Button>
            )}
          </div>

          {hasPath && (
            <div className="rounded-lg border border-border bg-muted/20 px-3 py-2">
              <div className="mb-1 flex items-center gap-2 text-xs">
                <span className={missing ? "text-amber-700 dark:text-amber-300" : "text-emerald-600 dark:text-emerald-400"}>
                  ● {missing ? t.detail.missingFromDisk : t.detail.onDisk}
                </span>
              </div>
              <div className="select-all break-all font-mono text-xs text-muted-foreground">{asset.local_path}</div>
            </div>
          )}

          {asset.description && (
            <Section title={t.detail.description}>
              <p className="whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">{asset.description}</p>
            </Section>
          )}

          {asset.tags.length > 0 && (
            <Section title={t.detail.tags}>
              <div className="flex flex-wrap gap-1.5">
                {asset.tags.map((t) => (
                  <Link
                    key={t}
                    href={`/?tags=${encodeURIComponent(t)}`}
                    className="rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground"
                  >
                    #{t}
                  </Link>
                ))}
              </div>
            </Section>
          )}

          {(asset.compatible_avatars?.length ?? 0) > 0 && (
            <Section title={t.detail.compatibleAvatars}>
              <div className="flex flex-wrap gap-1.5">
                {asset.compatible_avatars!.map((c) =>
                  c.avatar_asset_id !== null ? (
                    <Link
                      key={c.avatar_name}
                      href={avatarHref(c.avatar_asset_id)}
                      className="inline-flex items-center gap-1 rounded-md border border-violet-300 dark:border-violet-800/70 bg-violet-100 dark:bg-violet-950/40 px-2 py-0.5 text-xs text-violet-800 dark:text-violet-200 hover:border-violet-500"
                    >
                      <UserRound className="size-3.5" /> {c.avatar_name}
                    </Link>
                  ) : (
                    <span
                      key={c.avatar_name}
                      className="inline-flex items-center gap-1 rounded-md bg-muted px-2 py-0.5 text-xs text-muted-foreground"
                      title={t.common.notInLibrary}
                    >
                      <UserRound className="size-3.5" /> {c.avatar_name}
                    </span>
                  )
                )}
              </div>
            </Section>
          )}

          {isAvatar && (
            <Link href={avatarHref(asset.id)} className="inline-flex items-center gap-1.5 text-sm text-violet-700 dark:text-violet-300 hover:text-violet-800 dark:hover:text-violet-200">
              <UserRound className="size-4" /> {t.detail.everythingCompatible}
            </Link>
          )}

          <AssetFilesPanel asset={asset} onChanged={update} />
        </div>
      </div>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t.detail.deleteTitle(asset.name)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t.detail.deleteText}
              {hasPath && (
                <>
                  {" "}
                  {t.detail.filesUntouched(<span className="break-all font-mono">{asset.local_path}</span>)}
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy === "delete"}>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
                handleDelete();
              }}
              disabled={busy === "delete"}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {busy === "delete" ? t.common.deleting : t.common.delete}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
