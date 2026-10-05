"use client";

import React, { useState } from "react";
import { Asset, AssetFile, addAssetFile, deleteAssetFile, getAssetByID, pickFolder } from "@/lib/api";
import { File as FileIcon, FileArchive, Folder, FolderOpen, Package, X } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import type { Messages } from "@/lib/messages/en";

interface AssetFilesPanelProps {
  asset: Asset;
  /** Called with the reloaded asset after a file is added or removed. */
  onChanged: (asset: Asset) => void;
}

const kindLabel = (kind: AssetFile["kind"], t: Messages): React.ReactNode =>
  ({
    folder: <><Folder className="inline size-3.5" /> {t.files.folder}</>,
    archive: <><FileArchive className="inline size-3.5" /> {t.files.archive}</>,
    unitypackage: <><Package className="inline size-3.5" /> {t.files.package}</>,
    file: <><FileIcon className="inline size-3.5" /> {t.files.file}</>,
  })[kind];

/**
 * Lists every file/folder linked to an asset (extracted folder, archived zip,
 * older versions) and lets the user link or unlink more. Unlinking never
 * touches anything on disk.
 */
export const AssetFilesPanel: React.FC<AssetFilesPanelProps> = ({ asset, onChanged }) => {
  const files = asset.files ?? [];
  const [isAdding, setIsAdding] = useState(false);
  const [path, setPath] = useState("");
  const [version, setVersion] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { t } = useI18n();

  const reload = async () => onChanged(await getAssetByID(asset.id));

  const handleAdd = async () => {
    if (!path.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await addAssetFile(asset.id, { path: path.trim(), version: version.trim() });
      setPath("");
      setVersion("");
      setIsAdding(false);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.files.addFailed);
    } finally {
      setBusy(false);
    }
  };

  const handleRemove = async (file: AssetFile) => {
    setBusy(true);
    setError(null);
    try {
      await deleteAssetFile(asset.id, file.id);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t.files.removeFailed);
    } finally {
      setBusy(false);
    }
  };

  const handleBrowse = async () => {
    const selected = await pickFolder();
    if (selected) setPath(selected);
  };

  return (
    <div className="p-3.5 rounded-xl bg-background/80 border border-border space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs uppercase font-mono tracking-wider text-muted-foreground font-semibold">
          {t.files.title(files.length)}
        </span>
        {!isAdding && (
          <button
            type="button"
            onClick={() => setIsAdding(true)}
            className="text-xs text-primary hover:text-primary transition-colors cursor-pointer"
          >
            {t.files.link}
          </button>
        )}
      </div>

      {files.length === 0 && !isAdding && (
        <p className="text-xs text-muted-foreground italic">{t.files.none}</p>
      )}

      <ul className="space-y-1.5">
        {files.map((f) => (
          <li
            key={f.id}
            className="flex items-start gap-2 rounded-lg bg-card/60 border border-border p-2"
          >
            <div className="flex-1 min-w-0">
              <div className="flex flex-wrap items-center gap-1.5 text-xs">
                <span className="text-foreground">{kindLabel(f.kind, t)}</span>
                {f.version && (
                  <span className="px-1.5 rounded bg-primary/15 text-primary border border-primary/60 font-mono">
                    v{f.version}
                  </span>
                )}
                {f.path === asset.local_path && (
                  <span className="px-1.5 rounded bg-muted text-foreground border border-border">
                    {t.files.primary}
                  </span>
                )}
                <span className={f.exists ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}>
                  {f.exists ? t.files.onDisk : t.files.missing}
                </span>
              </div>
              <div className="mt-1 font-mono text-xs text-muted-foreground break-all select-all">{f.path}</div>
            </div>
            <button
              type="button"
              onClick={() => handleRemove(f)}
              disabled={busy}
              className="text-muted-foreground hover:text-rose-600 dark:hover:text-rose-400 text-xs p-1 cursor-pointer disabled:opacity-40"
              title={t.files.unlink}
            >
              <X className="size-4" />
            </button>
          </li>
        ))}
      </ul>

      {isAdding && (
        <div className="space-y-2 pt-1">
          <div className="flex gap-2">
            <input
              type="text"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              placeholder={t.files.pathPlaceholder}
              className="flex-1 min-w-0 font-mono rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-foreground placeholder-muted-foreground focus:border-ring focus:outline-none"
            />
            <button
              type="button"
              onClick={handleBrowse}
              className="px-2.5 py-1.5 rounded-lg bg-muted hover:bg-accent text-foreground text-xs border border-border cursor-pointer shrink-0"
              title={t.files.browse}
            >
              <FolderOpen className="size-4" />
            </button>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder={t.files.versionPlaceholder}
              className="flex-1 min-w-0 rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-foreground placeholder-muted-foreground focus:border-ring focus:outline-none"
            />
            <button
              type="button"
              onClick={handleAdd}
              disabled={busy || !path.trim()}
              className="px-3 py-1.5 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold cursor-pointer disabled:opacity-50"
            >
              {t.files.linkButton}
            </button>
            <button
              type="button"
              onClick={() => {
                setIsAdding(false);
                setError(null);
              }}
              className="px-3 py-1.5 rounded-lg bg-card text-muted-foreground hover:text-foreground text-xs border border-border cursor-pointer"
            >
              {t.common.cancel}
            </button>
          </div>
        </div>
      )}

      {error && <p className="text-xs text-rose-600 dark:text-rose-400">{error}</p>}
    </div>
  );
};
