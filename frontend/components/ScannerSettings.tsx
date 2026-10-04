"use client";

import React, { useState } from "react";
import { Category, ScannerConfig, buildCategoryTree, pickFolder, saveScannerConfig } from "@/lib/api";

interface ScannerSettingsProps {
  config: ScannerConfig;
  categories: Category[];
  onSaved: (config: ScannerConfig) => void;
}

const inputClass =
  "rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-1.5 text-xs text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500";
const labelClass = "block text-[11px] uppercase tracking-wider font-mono text-neutral-400 mb-1.5";

const toList = (text: string) =>
  text
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

/**
 * Edits where the scanner looks and how folder names map to categories.
 */
export const ScannerSettings: React.FC<ScannerSettingsProps> = ({ config, categories, onSaved }) => {
  const [roots, setRoots] = useState<string[]>(config.roots);
  const [rootInput, setRootInput] = useState("");
  const [archiveDirs, setArchiveDirs] = useState(config.archive_dirs.join(", "));
  const [ignore, setIgnore] = useState(config.ignore.join(", "));
  const [deps, setDeps] = useState(config.known_dependencies.join(", "));
  const [mapping, setMapping] = useState<[string, string][]>(
    Object.entries(config.folder_map).sort(([a], [b]) => a.localeCompare(b))
  );
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);

  const addRoot = (path: string) => {
    const p = path.trim();
    if (p && !roots.some((r) => r.toLowerCase() === p.toLowerCase())) {
      setRoots([...roots, p]);
    }
    setRootInput("");
  };

  const handleBrowse = async () => {
    const selected = await pickFolder();
    if (selected) addRoot(selected);
  };

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const saved = await saveScannerConfig({
        roots: rootInput.trim() ? [...roots, rootInput.trim()] : roots,
        archive_dirs: toList(archiveDirs),
        ignore: toList(ignore),
        known_dependencies: toList(deps),
        folder_map: Object.fromEntries(mapping.filter(([folder, cat]) => folder.trim() && cat)),
      });
      setRoots(saved.roots);
      setRootInput("");
      setMessage({ ok: true, text: "Settings saved." });
      onSaved(saved);
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : "Failed to save settings" });
    } finally {
      setSaving(false);
    }
  };

  const tree = buildCategoryTree(categories);

  return (
    <div className="space-y-5">
      {/* Library roots */}
      <div>
        <span className={labelClass}>Library folders</span>
        {roots.length === 0 && (
          <p className="text-xs text-amber-300 mb-2">Add the folder that holds your assets, e.g. N:\Unity Materials.</p>
        )}
        <ul className="space-y-1 mb-2">
          {roots.map((r) => (
            <li
              key={r}
              className="flex items-center gap-2 rounded-lg bg-neutral-950/80 border border-neutral-800 px-3 py-1.5"
            >
              <span className="flex-1 font-mono text-xs text-neutral-200 break-all">📁 {r}</span>
              <button
                type="button"
                onClick={() => setRoots(roots.filter((x) => x !== r))}
                className="text-neutral-500 hover:text-rose-400 text-xs cursor-pointer"
                title="Remove"
              >
                ✕
              </button>
            </li>
          ))}
        </ul>
        <div className="flex gap-2">
          <input
            value={rootInput}
            onChange={(e) => setRootInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                addRoot(rootInput);
              }
            }}
            placeholder="Folder path, then Enter"
            className={`${inputClass} flex-1 font-mono`}
          />
          <button
            type="button"
            onClick={handleBrowse}
            className="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs border border-neutral-700 cursor-pointer"
          >
            📂 Browse
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label className={labelClass}>Archive folders</label>
          <input value={archiveDirs} onChange={(e) => setArchiveDirs(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-[11px] text-neutral-500">Where original zips are kept (comma separated).</p>
        </div>
        <div>
          <label className={labelClass}>Never scan</label>
          <input value={ignore} onChange={(e) => setIgnore(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-[11px] text-neutral-500">Folder/file names skipped and never read.</p>
        </div>
        <div>
          <label className={labelClass}>Dependency BOOTH ids</label>
          <input value={deps} onChange={(e) => setDeps(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-[11px] text-neutral-500">Linked in readmes but never the asset (lilToon…).</p>
        </div>
      </div>

      {/* Folder -> category mapping */}
      <div>
        <span className={labelClass}>Folder name → category</span>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-1.5 max-h-72 overflow-y-auto pr-1">
          {mapping.map(([folder, category], idx) => (
            <div key={idx} className="flex items-center gap-1.5">
              <input
                value={folder}
                onChange={(e) =>
                  setMapping(mapping.map((m, i) => (i === idx ? [e.target.value.toLowerCase(), m[1]] : m)))
                }
                placeholder="folder"
                className={`${inputClass} w-28 font-mono`}
              />
              <span className="text-neutral-600 text-xs">→</span>
              <select
                value={category}
                onChange={(e) => setMapping(mapping.map((m, i) => (i === idx ? [m[0], e.target.value] : m)))}
                className={`${inputClass} flex-1 min-w-0 cursor-pointer`}
              >
                {!categories.some((c) => c.name === category) && <option value={category}>{category} (missing)</option>}
                {tree.map((root) => (
                  <React.Fragment key={root.id}>
                    <option value={root.name}>{root.name}</option>
                    {root.children.map((child) => (
                      <option key={child.id} value={child.name}>
                        &nbsp;&nbsp;└ {child.name}
                      </option>
                    ))}
                  </React.Fragment>
                ))}
              </select>
              <button
                type="button"
                onClick={() => setMapping(mapping.filter((_, i) => i !== idx))}
                className="text-neutral-500 hover:text-rose-400 text-xs px-1 cursor-pointer"
                title="Remove"
              >
                ✕
              </button>
            </div>
          ))}
        </div>
        <button
          type="button"
          onClick={() => setMapping([...mapping, ["", categories[0]?.name ?? "Other"]])}
          className="mt-2 text-[11px] text-cyan-400 hover:text-cyan-300 cursor-pointer"
        >
          + Add mapping
        </button>
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold cursor-pointer disabled:opacity-50"
        >
          {saving ? "Saving…" : "Save settings"}
        </button>
        {message && <span className={`text-xs ${message.ok ? "text-emerald-400" : "text-rose-400"}`}>{message.text}</span>}
      </div>
    </div>
  );
};
