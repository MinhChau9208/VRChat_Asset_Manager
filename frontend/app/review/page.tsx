"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Asset,
  Category,
  ScanResult,
  ScannerConfig,
  acceptDrafts,
  applyBooth,
  BoothApplyResult,
  boothSearchUrl,
  assetUpdatePayload,
  buildCategoryTree,
  getAssetByID,
  getAssetPreviewUrl,
  getAssets,
  getCategories,
  getIgnoredPaths,
  getScannerConfig,
  ignoreDrafts,
  runScan,
  unignorePath,
  updateAsset,
} from "@/lib/api";
import { ScannerSettings } from "@/components/ScannerSettings";
import { Check, Package, ScanSearch, Settings2, ShoppingBag, TriangleAlert, UserRound, X } from "lucide-react";
import { assetHref } from "@/lib/routes";
import { useI18n } from "@/lib/i18n";
import type { Messages } from "@/lib/messages/en";

const selectClass =
  "rounded-lg border border-border bg-background/80 px-2 py-1 text-xs text-foreground focus:border-ring focus:outline-none cursor-pointer";

function CategorySelect({
  categories,
  value,
  onChange,
  placeholder,
}: {
  categories: Category[];
  value: number | null;
  onChange: (id: number | null) => void;
  placeholder?: string;
}) {
  const { t, categoryName } = useI18n();
  return (
    <select
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value === "" ? null : Number(e.target.value))}
      className={selectClass}
    >
      <option value="">{placeholder ?? t.review.noCategory}</option>
      {buildCategoryTree(categories).map((root) => (
        <React.Fragment key={root.id}>
          <option value={root.id}>{categoryName(root.name)}</option>
          {root.children.map((child) => (
            <option key={child.id} value={child.id}>
              &nbsp;&nbsp;└ {categoryName(child.name)}
            </option>
          ))}
        </React.Fragment>
      ))}
    </select>
  );
}

// The backend reports changed fields as "name", "compatible:Manuka", "tag:…".
function changeLabel(change: string, t: Messages): string {
  if (change.startsWith("compatible:")) return t.review.changedCompat(change.slice("compatible:".length));
  if (change.startsWith("tag:")) return t.review.changedTag(change.slice("tag:".length));
  const fields: Record<string, string> = {
    name: t.review.changedName,
    author: t.review.changedAuthor,
    category: t.review.changedCategory,
    preview: t.review.changedPreview,
  };
  return fields[change] ?? change;
}

export default function ReviewPage() {
  const { t } = useI18n();
  const [config, setConfig] = useState<ScannerConfig | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [drafts, setDrafts] = useState<Asset[]>([]);
  const [missingCount, setMissingCount] = useState(0);
  const [ignoredPaths, setIgnoredPaths] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [reloadToken, setReloadToken] = useState(0);

  const [showSettings, setShowSettings] = useState(false);
  const [showIgnored, setShowIgnored] = useState(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [scanResult, setScanResult] = useState<ScanResult | null>(null);
  const [boothResults, setBoothResults] = useState<BoothApplyResult[] | null>(null);
  const [isScanning, setIsScanning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      getScannerConfig(),
      getCategories(),
      getAssets({ status: "draft", sort: "name_asc" }),
      getAssets({ local_status: "missing" }),
      getIgnoredPaths(),
    ])
      // The list endpoint omits files and compatibility, so load each draft fully.
      .then(async ([cfg, cats, draftList, missing, ignored]) => {
        const full = await Promise.all(draftList.map((d) => getAssetByID(d.id)));
        if (cancelled) return;
        setConfig(cfg);
        setCategories(cats);
        setDrafts(full);
        setMissingCount(missing.length);
        setIgnoredPaths(ignored);
        setSelected((prev) => new Set([...prev].filter((id) => full.some((d) => d.id === id))));
        if (cfg.roots.length === 0) setShowSettings(true);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : t.review.loadFailed);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [reloadToken, t]);

  const reload = () => setReloadToken((t) => t + 1);

  // Runs a mutation and reloads; errors are shown in the banner.
  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.somethingWrong);
    } finally {
      setBusy(false);
    }
  };

  const handleScan = async () => {
    setIsScanning(true);
    setError(null);
    try {
      setScanResult(await runScan());
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.review.scanFailed);
    } finally {
      setIsScanning(false);
    }
  };

  const toggle = (id: number) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const selectedIds = [...selected];
  const allSelected = drafts.length > 0 && selected.size === drafts.length;

  const handleIgnore = (ids: number[]) => {
    const message = ids.length === 1 ? t.review.ignoreOne : t.review.ignoreMany(ids.length);
    if (window.confirm(message)) run(() => ignoreDrafts(ids));
  };

  const setCategory = (ids: number[], categoryId: number | null) =>
    run(async () => {
      for (const id of ids) {
        const d = drafts.find((x) => x.id === id);
        if (d) await updateAsset(id, assetUpdatePayload(d, { category_id: categoryId }));
      }
    });

  // Fills drafts from BOOTH (one request per second, so this can take a moment).
  const handleFetchBooth = (ids: number[]) => {
    const withLink = ids.filter((id) => drafts.find((d) => d.id === id)?.booth_url);
    if (withLink.length === 0) {
      setError(t.review.noBoothLinks);
      return;
    }
    run(async () => {
      const { results } = await applyBooth(withLink);
      setBoothResults(results);
    });
  };

  const sourceLabel = (source: string) => (source === "folder name" ? t.review.folderName : source);

  const setBooth = (d: Asset, url: string) => run(() => updateAsset(d.id, assetUpdatePayload(d, { booth_url: url })));

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="max-w-6xl mx-auto px-4 py-8 space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
          <div>
            <Link href="/" className="text-xs text-muted-foreground hover:text-primary transition-colors">
              {t.common.backToLibrary}
            </Link>
            <h1 className="mt-1 text-xl font-bold tracking-tight text-foreground">{t.review.title}</h1>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t.review.intro}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowSettings((v) => !v)}
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-card hover:bg-muted text-foreground text-xs border border-border cursor-pointer"
            >
              <Settings2 className="size-4" /> {t.review.settings}
            </button>
            <button
              type="button"
              onClick={handleScan}
              disabled={isScanning || !config || config.roots.length === 0}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isScanning ? (
                <>
                  <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  {t.review.scanning}
                </>
              ) : (
                <><ScanSearch className="size-4" /> {t.review.scanNow}</>
              )}
            </button>
          </div>
        </div>

        {error && (
          <div className="p-3 rounded-lg bg-rose-100 dark:bg-rose-950/40 border border-rose-300 dark:border-rose-800/60 text-xs text-rose-700 dark:text-rose-300">{error}</div>
        )}

        {showSettings && config && (
          <section className="p-5 rounded-xl bg-card/60 border border-border">
            <ScannerSettings
              key={JSON.stringify(config)}
              config={config}
              categories={categories}
              onSaved={(saved) => setConfig(saved)}
            />
          </section>
        )}

        {/* Last scan result */}
        {scanResult && (
          <section className="p-4 rounded-xl bg-card/60 border border-border text-xs space-y-2">
            <div className="flex flex-wrap gap-x-5 gap-y-1 text-foreground">
              <span>{t.review.newDrafts(<b className="text-foreground">{scanResult.created}</b>)}</span>
              <span>{t.review.attached(<b className="text-foreground">{scanResult.attached.length}</b>)}</span>
              <span>{t.review.alreadyLinked(<b className="text-foreground">{scanResult.already_linked}</b>)}</span>
              {scanResult.ignored > 0 && (
                <span>{t.review.ignoredSkipped(<b className="text-foreground">{scanResult.ignored}</b>)}</span>
              )}
              <span className="text-muted-foreground">{scanResult.duration_ms} ms</span>
            </div>
            {scanResult.attached.length > 0 && (
              <details>
                <summary className="cursor-pointer text-primary">{t.review.showLinked}</summary>
                <ul className="mt-2 space-y-0.5 font-mono text-xs text-muted-foreground">
                  {scanResult.attached.map((a) => (
                    <li key={a.path}>
                      <Link href={assetHref(a.asset_id)} className="text-foreground hover:text-primary">
                        {a.asset_name}
                      </Link>{" "}
                      ← {a.path}
                    </li>
                  ))}
                </ul>
              </details>
            )}
            {scanResult.warnings.length > 0 && (
              <ul className="text-amber-700 dark:text-amber-300 space-y-0.5">
                {scanResult.warnings.map((w) => (
                  <li key={w} className="flex items-center gap-1.5"><TriangleAlert className="size-3.5 shrink-0" /> {w}</li>
                ))}
              </ul>
            )}
          </section>
        )}

        {boothResults && (
          <section className="p-4 rounded-xl bg-card/60 border border-red-300 dark:border-red-900/40 text-xs space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-foreground">
                {t.review.boothUpdated(<b className="text-foreground">{boothResults.filter((r) => r.ok).length}</b>)}
                {boothResults.some((r) => !r.ok) &&
                  t.review.boothFailed(
                    <b className="text-rose-700 dark:text-rose-300">{boothResults.filter((r) => !r.ok).length}</b>
                  )}
              </span>
              <button type="button" onClick={() => setBoothResults(null)} className="text-muted-foreground hover:text-foreground cursor-pointer">
                <X className="size-4" />
              </button>
            </div>
            <ul className="space-y-0.5 text-xs">
              {boothResults.map((r) => (
                <li key={r.asset_id} className={r.ok ? "text-muted-foreground" : "text-rose-700 dark:text-rose-300"}>
                  {r.ok ? <Check className="inline size-3.5" /> : <X className="inline size-3.5" />} {r.name}
                  {r.ok && r.changed && r.changed.length > 0 && <span className="text-muted-foreground"> · {r.changed.map((c) => changeLabel(c, t)).join(", ")}</span>}
                  {r.error && <span> · {r.error}</span>}
                </li>
              ))}
            </ul>
          </section>
        )}

        {missingCount > 0 && (
          <Link
            href="/?local_status=missing"
            className="block p-3 rounded-lg bg-amber-100 dark:bg-amber-950/30 border border-amber-300 dark:border-amber-800/50 text-xs text-amber-700 dark:text-amber-300 hover:border-amber-600"
          >
            <TriangleAlert className="mr-1 inline size-4" /> {t.review.missing(missingCount)}
          </Link>
        )}

        {/* Bulk actions */}
        <div className="flex flex-wrap items-center gap-2 sticky top-0 z-10 py-2 bg-background/90 backdrop-blur">
          <label className="inline-flex items-center gap-2 text-xs text-foreground cursor-pointer mr-2">
            <input
              type="checkbox"
              checked={allSelected}
              onChange={() => setSelected(allSelected ? new Set() : new Set(drafts.map((d) => d.id)))}
              className="accent-cyan-500"
            />
            {t.review.drafts(drafts.length)}
            {selected.size > 0 && t.review.selectedSuffix(selected.size)}
          </label>
          {selected.size > 0 && (
            <>
              <button
                type="button"
                onClick={() => run(() => acceptDrafts(selectedIds))}
                disabled={busy}
                className="px-3 py-1.5 rounded-lg bg-emerald-700 hover:bg-emerald-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
              >
                <Check className="size-4" /> {t.review.acceptSelected}
              </button>
              <button
                type="button"
                onClick={() => handleFetchBooth(selectedIds)}
                disabled={busy}
                className="px-3 py-1.5 rounded-lg bg-red-600 dark:bg-red-800/80 hover:bg-red-700 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
                title={t.review.fetchTitle}
              >
                <ShoppingBag className="size-4" /> {busy ? t.review.working : t.review.fetchBooth}
              </button>
              <button
                type="button"
                onClick={() => handleIgnore(selectedIds)}
                disabled={busy}
                className="px-3 py-1.5 rounded-lg bg-muted hover:bg-accent text-foreground text-xs border border-border cursor-pointer disabled:opacity-50"
              >
                {t.review.ignoreSelected}
              </button>
              <CategorySelect
                categories={categories}
                value={null}
                onChange={(id) => setCategory(selectedIds, id)}
                placeholder={t.review.setCategory}
              />
            </>
          )}
        </div>

        {/* Draft list */}
        {isLoading ? (
          <p className="text-xs text-muted-foreground">{t.common.loading}</p>
        ) : drafts.length === 0 ? (
          <div className="p-10 text-center rounded-xl border border-dashed border-border text-sm text-muted-foreground">
            {t.review.noDrafts} {config?.roots.length ? t.review.runScan : t.review.addFolderFirst}
          </div>
        ) : (
          <ul className="space-y-2">
            {drafts.map((d) => {
              const info = d.scan_info;
              const versions = [...new Set((d.files ?? []).map((f) => f.version).filter(Boolean))];
              const otherCandidates = (info?.booth_candidates ?? []).filter((c) => c.url !== d.booth_url);
              return (
                <li
                  key={d.id}
                  className={`flex gap-3 p-3 rounded-xl border transition-colors ${
                    selected.has(d.id) ? "bg-primary/20 border-primary/60" : "bg-card/60 border-border"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selected.has(d.id)}
                    onChange={() => toggle(d.id)}
                    className="mt-1 accent-cyan-500 shrink-0"
                  />
                  <div className="h-20 w-20 shrink-0 rounded-lg bg-background border border-border overflow-hidden flex items-center justify-center">
                    {d.preview_path ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={getAssetPreviewUrl(d.id, d.updated_at)} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <Package className="size-7 text-muted-foreground/40" strokeWidth={1.5} />
                    )}
                  </div>

                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link href={assetHref(d.id)} className="text-sm font-semibold text-foreground hover:text-primary">
                        {d.name}
                      </Link>
                      <CategorySelect
                        categories={categories}
                        value={d.category_id}
                        onChange={(id) => setCategory([d.id], id)}
                      />
                    </div>

                    <div className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                      <span title={(d.files ?? []).map((f) => f.path).join("\n")}>
                        {t.review.files((d.files ?? []).length)}
                      </span>
                      {versions.map((v) => (
                        <span key={v} className="px-1.5 rounded bg-primary/15 text-primary border border-primary/60 font-mono">
                          v{v}
                        </span>
                      ))}
                      <span className="font-mono text-muted-foreground truncate max-w-full">{d.local_path}</span>
                    </div>

                    <div className="flex flex-wrap items-center gap-1.5 text-xs">
                      {d.booth_url ? (
                        <a
                          href={d.booth_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="px-1.5 py-0.5 rounded bg-red-100 dark:bg-red-950/50 text-red-700 dark:text-red-300 border border-red-300 dark:border-red-800/60 hover:border-red-600"
                          title={info?.booth_source ? t.review.foundIn(sourceLabel(info.booth_source)) : undefined}
                        >
                          <ShoppingBag className="inline size-3.5" /> BOOTH {d.booth_url.split("/").pop()}
                          {info?.booth_source && <span className="text-red-600 dark:text-red-400/70"> · {sourceLabel(info.booth_source)}</span>}
                        </a>
                      ) : (
                        <a
                          href={boothSearchUrl(d.name)}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-muted-foreground hover:text-red-700 dark:hover:text-red-300"
                          title={t.review.searchTitle}
                        >
                          {t.review.noBoothLink}
                        </a>
                      )}
                      {d.booth_url && (
                        <button
                          type="button"
                          onClick={() => handleFetchBooth([d.id])}
                          disabled={busy}
                          className="px-1.5 py-0.5 rounded bg-muted text-red-800 dark:text-red-200 border border-red-300 dark:border-red-900/60 hover:border-red-600 cursor-pointer disabled:opacity-50"
                          title={t.review.fetchTitle}
                        >
                          {t.review.fetchInfo}
                        </button>
                      )}
                      {otherCandidates.map((c) => (
                        <button
                          key={c.url}
                          type="button"
                          onClick={() => setBooth(d, c.url)}
                          disabled={busy}
                          className="px-1.5 py-0.5 rounded bg-muted text-foreground border border-border hover:border-border cursor-pointer"
                          title={t.review.candidateTitle(sourceLabel(c.source))}
                        >
                          {t.review.useCandidate(c.url.split("/").pop() ?? "")}
                        </button>
                      ))}
                      {(d.compatible_avatars ?? []).map((c) => (
                        <span
                          key={c.avatar_name}
                          className="px-1.5 py-0.5 rounded bg-violet-100 dark:bg-violet-950/50 text-violet-800 dark:text-violet-200 border border-violet-300 dark:border-violet-800/70"
                          title={(info?.compat_reasons ?? []).join("\n")}
                        >
                          <UserRound className="inline size-3.5" /> {c.avatar_name}
                        </span>
                      ))}
                    </div>
                  </div>

                  <div className="flex flex-col gap-1.5 shrink-0">
                    <button
                      type="button"
                      onClick={() => run(() => acceptDrafts([d.id]))}
                      disabled={busy}
                      className="px-3 py-1 rounded-lg bg-emerald-700 hover:bg-emerald-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
                    >
                      <Check className="inline size-3.5" /> {t.common.accept}
                    </button>
                    <Link
                      href={assetHref(d.id)}
                      className="px-3 py-1 rounded-lg bg-muted hover:bg-accent text-foreground text-xs text-center border border-border"
                    >
                      {t.common.edit}
                    </Link>
                    <button
                      type="button"
                      onClick={() => handleIgnore([d.id])}
                      disabled={busy}
                      className="px-3 py-1 rounded-lg text-muted-foreground hover:text-rose-700 dark:hover:text-rose-300 text-xs cursor-pointer disabled:opacity-50"
                    >
                      {t.common.ignore}
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        )}

        {/* Ignored paths */}
        {ignoredPaths.length > 0 && (
          <section className="text-xs">
            <button
              type="button"
              onClick={() => setShowIgnored((v) => !v)}
              className="text-muted-foreground hover:text-foreground cursor-pointer"
            >
              {showIgnored ? "▾" : "▸"} {t.review.ignoredPaths(ignoredPaths.length)}
            </button>
            {showIgnored && (
              <ul className="mt-2 space-y-1">
                {ignoredPaths.map((p) => (
                  <li key={p} className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
                    <span className="flex-1 break-all">{p}</span>
                    <button
                      type="button"
                      onClick={() => run(() => unignorePath(p))}
                      disabled={busy}
                      className="text-primary hover:text-primary cursor-pointer font-sans"
                    >
                      {t.review.restore}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
