"use client";

import React, { useState } from "react";
import { Asset, AssetFile, addAssetFile, deleteAssetFile, getAssetByID, pickFolder } from "@/lib/api";
import { File as FileIcon, FileArchive, Folder, FolderOpen, Package, X } from "lucide-react";

interface AssetFilesPanelProps {
  asset: Asset;
  /** Called with the reloaded asset after a file is added or removed. */
  onChanged: (asset: Asset) => void;
}

const KIND_LABELS: Record<AssetFile["kind"], React.ReactNode> = {
  folder: <><Folder className="inline size-3.5" /> Folder</>,
  archive: <><FileArchive className="inline size-3.5" /> Archive</>,
  unitypackage: <><Package className="inline size-3.5" /> Package</>,
  file: <><FileIcon className="inline size-3.5" /> File</>,
};

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
      setError(err instanceof Error ? err.message : "Failed to add file");
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
      setError(err instanceof Error ? err.message : "Failed to remove file");
    } finally {
      setBusy(false);
    }
  };

  const handleBrowse = async () => {
    const selected = await pickFolder();
    if (selected) setPath(selected);
  };

  return (
    <div className="p-3.5 rounded-xl bg-neutral-950/80 border border-neutral-800 space-y-2.5">
      <div className="flex items-center justify-between">
        <span className="text-xs uppercase font-mono tracking-wider text-neutral-400 font-semibold">
          Files & Versions ({files.length})
        </span>
        {!isAdding && (
          <button
            type="button"
            onClick={() => setIsAdding(true)}
            className="text-xs text-cyan-400 hover:text-cyan-300 transition-colors cursor-pointer"
          >
            + Link file
          </button>
        )}
      </div>

      {files.length === 0 && !isAdding && (
        <p className="text-xs text-neutral-500 italic">No files linked.</p>
      )}

      <ul className="space-y-1.5">
        {files.map((f) => (
          <li
            key={f.id}
            className="flex items-start gap-2 rounded-lg bg-neutral-900/60 border border-neutral-800 p-2"
          >
            <div className="flex-1 min-w-0">
              <div className="flex flex-wrap items-center gap-1.5 text-xs">
                <span className="text-neutral-300">{KIND_LABELS[f.kind]}</span>
                {f.version && (
                  <span className="px-1.5 rounded bg-cyan-950/60 text-cyan-300 border border-cyan-800/60 font-mono">
                    v{f.version}
                  </span>
                )}
                {f.path === asset.local_path && (
                  <span className="px-1.5 rounded bg-neutral-800 text-neutral-300 border border-neutral-700">
                    primary
                  </span>
                )}
                <span className={f.exists ? "text-emerald-400" : "text-rose-400"}>
                  {f.exists ? "● on disk" : "● missing"}
                </span>
              </div>
              <div className="mt-1 font-mono text-xs text-neutral-400 break-all select-all">{f.path}</div>
            </div>
            <button
              type="button"
              onClick={() => handleRemove(f)}
              disabled={busy}
              className="text-neutral-500 hover:text-rose-400 text-xs p-1 cursor-pointer disabled:opacity-40"
              title="Unlink (files on disk are not touched)"
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
              placeholder="Folder, .zip or .unitypackage path"
              className="flex-1 min-w-0 font-mono rounded-lg border border-neutral-800 bg-neutral-900 px-2.5 py-1.5 text-xs text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none"
            />
            <button
              type="button"
              onClick={handleBrowse}
              className="px-2.5 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs border border-neutral-700 cursor-pointer shrink-0"
              title="Browse for a folder"
            >
              <FolderOpen className="size-4" />
            </button>
          </div>
          <div className="flex gap-2">
            <input
              type="text"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="Version (optional, e.g. 1.2.0)"
              className="flex-1 min-w-0 rounded-lg border border-neutral-800 bg-neutral-900 px-2.5 py-1.5 text-xs text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none"
            />
            <button
              type="button"
              onClick={handleAdd}
              disabled={busy || !path.trim()}
              className="px-3 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
            >
              Link
            </button>
            <button
              type="button"
              onClick={() => {
                setIsAdding(false);
                setError(null);
              }}
              className="px-3 py-1.5 rounded-lg bg-neutral-900 text-neutral-400 hover:text-neutral-200 text-xs border border-neutral-800 cursor-pointer"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {error && <p className="text-xs text-rose-400">{error}</p>}
    </div>
  );
};
