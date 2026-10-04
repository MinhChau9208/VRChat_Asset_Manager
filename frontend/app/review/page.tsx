"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Asset,
  Category,
  ScanResult,
  ScannerConfig,
  acceptDrafts,
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

const selectClass =
  "rounded-lg border border-neutral-800 bg-neutral-950/80 px-2 py-1 text-xs text-white focus:border-cyan-500 focus:outline-none cursor-pointer";

function CategorySelect({
  categories,
  value,
  onChange,
  placeholder = "No category",
}: {
  categories: Category[];
  value: number | null;
  onChange: (id: number | null) => void;
  placeholder?: string;
}) {
  return (
    <select
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value === "" ? null : Number(e.target.value))}
      className={selectClass}
    >
      <option value="">{placeholder}</option>
      {buildCategoryTree(categories).map((root) => (
        <React.Fragment key={root.id}>
          <option value={root.id}>{root.name}</option>
          {root.children.map((child) => (
            <option key={child.id} value={child.id}>
              &nbsp;&nbsp;└ {child.name}
            </option>
          ))}
        </React.Fragment>
      ))}
    </select>
  );
}

export default function ReviewPage() {
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
        if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load review data");
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [reloadToken]);

  const reload = () => setReloadToken((t) => t + 1);

  // Runs a mutation and reloads; errors are shown in the banner.
  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
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
      setError(err instanceof Error ? err.message : "Scan failed");
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
    const message =
      ids.length === 1
        ? "Ignore this draft? It is removed and later scans skip its files (nothing on disk is touched)."
        : `Ignore ${ids.length} drafts? They are removed and later scans skip their files (nothing on disk is touched).`;
    if (window.confirm(message)) run(() => ignoreDrafts(ids));
  };

  const setCategory = (ids: number[], categoryId: number | null) =>
    run(async () => {
      for (const id of ids) {
        const d = drafts.find((x) => x.id === id);
        if (d) await updateAsset(id, assetUpdatePayload(d, { category_id: categoryId }));
      }
    });

  const setBooth = (d: Asset, url: string) => run(() => updateAsset(d.id, assetUpdatePayload(d, { booth_url: url })));

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100">
      <div className="max-w-6xl mx-auto px-4 py-8 space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
          <div>
            <Link href="/" className="text-xs text-neutral-400 hover:text-cyan-400 transition-colors">
              ← Back to library
            </Link>
            <h1 className="mt-1 text-xl font-bold tracking-tight text-white">Scan & Review</h1>
            <p className="text-xs text-neutral-400 mt-0.5">
              Scanning only reads your folders. New assets arrive as drafts for you to accept, edit or ignore.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setShowSettings((v) => !v)}
              className="px-3 py-2 rounded-lg bg-neutral-900 hover:bg-neutral-800 text-neutral-300 text-xs border border-neutral-800 cursor-pointer"
            >
              ⚙️ Settings
            </button>
            <button
              type="button"
              onClick={handleScan}
              disabled={isScanning || !config || config.roots.length === 0}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isScanning ? (
                <>
                  <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  Scanning…
                </>
              ) : (
                "🔍 Scan now"
              )}
            </button>
          </div>
        </div>

        {error && (
          <div className="p-3 rounded-lg bg-rose-950/40 border border-rose-800/60 text-xs text-rose-300">{error}</div>
        )}

        {showSettings && config && (
          <section className="p-5 rounded-xl bg-neutral-900/60 border border-neutral-800">
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
          <section className="p-4 rounded-xl bg-neutral-900/60 border border-neutral-800 text-xs space-y-2">
            <div className="flex flex-wrap gap-x-5 gap-y-1 text-neutral-300">
              <span>
                <b className="text-white">{scanResult.created}</b> new drafts
              </span>
              <span>
                <b className="text-white">{scanResult.attached.length}</b> files linked to existing assets
              </span>
              <span>
                <b className="text-white">{scanResult.already_linked}</b> already in library
              </span>
              {scanResult.ignored > 0 && (
                <span>
                  <b className="text-white">{scanResult.ignored}</b> ignored paths skipped
                </span>
              )}
              <span className="text-neutral-500">{scanResult.duration_ms} ms</span>
            </div>
            {scanResult.attached.length > 0 && (
              <details>
                <summary className="cursor-pointer text-cyan-400">Show linked files</summary>
                <ul className="mt-2 space-y-0.5 font-mono text-[11px] text-neutral-400">
                  {scanResult.attached.map((a) => (
                    <li key={a.path}>
                      <Link href={`/assets/${a.asset_id}`} className="text-neutral-200 hover:text-cyan-300">
                        {a.asset_name}
                      </Link>{" "}
                      ← {a.path}
                    </li>
                  ))}
                </ul>
              </details>
            )}
            {scanResult.warnings.length > 0 && (
              <ul className="text-amber-300 space-y-0.5">
                {scanResult.warnings.map((w) => (
                  <li key={w}>⚠ {w}</li>
                ))}
              </ul>
            )}
          </section>
        )}

        {missingCount > 0 && (
          <Link
            href="/?local_status=missing"
            className="block p-3 rounded-lg bg-amber-950/30 border border-amber-800/50 text-xs text-amber-300 hover:border-amber-600"
          >
            ⚠ {missingCount} asset{missingCount === 1 ? "" : "s"} in your library point to files that are missing
            from disk. View them →
          </Link>
        )}

        {/* Bulk actions */}
        <div className="flex flex-wrap items-center gap-2 sticky top-0 z-10 py-2 bg-neutral-950/90 backdrop-blur">
          <label className="inline-flex items-center gap-2 text-xs text-neutral-300 cursor-pointer mr-2">
            <input
              type="checkbox"
              checked={allSelected}
              onChange={() => setSelected(allSelected ? new Set() : new Set(drafts.map((d) => d.id)))}
              className="accent-cyan-500"
            />
            {drafts.length} draft{drafts.length === 1 ? "" : "s"}
            {selected.size > 0 && ` · ${selected.size} selected`}
          </label>
          {selected.size > 0 && (
            <>
              <button
                type="button"
                onClick={() => run(() => acceptDrafts(selectedIds))}
                disabled={busy}
                className="px-3 py-1.5 rounded-lg bg-emerald-700 hover:bg-emerald-600 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
              >
                ✓ Accept selected
              </button>
              <button
                type="button"
                onClick={() => handleIgnore(selectedIds)}
                disabled={busy}
                className="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs border border-neutral-700 cursor-pointer disabled:opacity-50"
              >
                Ignore selected
              </button>
              <CategorySelect
                categories={categories}
                value={null}
                onChange={(id) => setCategory(selectedIds, id)}
                placeholder="Set category…"
              />
            </>
          )}
        </div>

        {/* Draft list */}
        {isLoading ? (
          <p className="text-xs text-neutral-500">Loading…</p>
        ) : drafts.length === 0 ? (
          <div className="p-10 text-center rounded-xl border border-dashed border-neutral-800 text-sm text-neutral-400">
            No drafts to review. {config?.roots.length ? "Run a scan to look for new assets." : "Add a library folder in Settings first."}
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
                    selected.has(d.id) ? "bg-cyan-950/20 border-cyan-800/60" : "bg-neutral-900/60 border-neutral-800"
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={selected.has(d.id)}
                    onChange={() => toggle(d.id)}
                    className="mt-1 accent-cyan-500 shrink-0"
                  />
                  <div className="h-20 w-20 shrink-0 rounded-lg bg-neutral-950 border border-neutral-800 overflow-hidden flex items-center justify-center">
                    {d.preview_path ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={getAssetPreviewUrl(d.id, d.updated_at)} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <span className="text-2xl opacity-40">📦</span>
                    )}
                  </div>

                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link href={`/assets/${d.id}`} className="text-sm font-semibold text-white hover:text-cyan-300">
                        {d.name}
                      </Link>
                      <CategorySelect
                        categories={categories}
                        value={d.category_id}
                        onChange={(id) => setCategory([d.id], id)}
                      />
                    </div>

                    <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-neutral-400">
                      <span title={(d.files ?? []).map((f) => f.path).join("\n")}>
                        {(d.files ?? []).length} file{(d.files ?? []).length === 1 ? "" : "s"}
                      </span>
                      {versions.map((v) => (
                        <span key={v} className="px-1.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-mono">
                          v{v}
                        </span>
                      ))}
                      <span className="font-mono text-neutral-500 truncate max-w-full">{d.local_path}</span>
                    </div>

                    <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                      {d.booth_url ? (
                        <a
                          href={d.booth_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="px-1.5 py-0.5 rounded bg-red-950/50 text-red-300 border border-red-800/60 hover:border-red-600"
                          title={info?.booth_source ? `Found in ${info.booth_source}` : undefined}
                        >
                          🛒 BOOTH {d.booth_url.split("/").pop()}
                          {info?.booth_source && <span className="text-red-400/70"> · {info.booth_source}</span>}
                        </a>
                      ) : (
                        <span className="text-neutral-500">No BOOTH link found</span>
                      )}
                      {otherCandidates.map((c) => (
                        <button
                          key={c.url}
                          type="button"
                          onClick={() => setBooth(d, c.url)}
                          disabled={busy}
                          className="px-1.5 py-0.5 rounded bg-neutral-800 text-neutral-300 border border-neutral-700 hover:border-neutral-500 cursor-pointer"
                          title={`Found in ${c.source}. Click to use.`}
                        >
                          use {c.url.split("/").pop()}?
                        </button>
                      ))}
                      {(d.compatible_avatars ?? []).map((c) => (
                        <span
                          key={c.avatar_name}
                          className="px-1.5 py-0.5 rounded bg-violet-950/50 text-violet-200 border border-violet-800/70"
                          title={(info?.compat_reasons ?? []).join("\n")}
                        >
                          👤 {c.avatar_name}
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
                      ✓ Accept
                    </button>
                    <Link
                      href={`/assets/${d.id}`}
                      className="px-3 py-1 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs text-center border border-neutral-700"
                    >
                      Edit
                    </Link>
                    <button
                      type="button"
                      onClick={() => handleIgnore([d.id])}
                      disabled={busy}
                      className="px-3 py-1 rounded-lg text-neutral-400 hover:text-rose-300 text-xs cursor-pointer disabled:opacity-50"
                    >
                      Ignore
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
              className="text-neutral-400 hover:text-neutral-200 cursor-pointer"
            >
              {showIgnored ? "▾" : "▸"} {ignoredPaths.length} ignored path{ignoredPaths.length === 1 ? "" : "s"}
            </button>
            {showIgnored && (
              <ul className="mt-2 space-y-1">
                {ignoredPaths.map((p) => (
                  <li key={p} className="flex items-center gap-2 font-mono text-[11px] text-neutral-500">
                    <span className="flex-1 break-all">{p}</span>
                    <button
                      type="button"
                      onClick={() => run(() => unignorePath(p))}
                      disabled={busy}
                      className="text-cyan-400 hover:text-cyan-300 cursor-pointer font-sans"
                    >
                      Restore
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
