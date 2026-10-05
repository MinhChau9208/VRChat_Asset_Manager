"use client";

import React, { useState } from "react";
import { Category, ScannerConfig, buildCategoryTree, pickFolder, saveScannerConfig } from "@/lib/api";
import { Folder, FolderOpen, X } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface ScannerSettingsProps {
  config: ScannerConfig;
  categories: Category[];
  onSaved: (config: ScannerConfig) => void;
}

const inputClass =
  "rounded-lg border border-border bg-background/80 px-3 py-1.5 text-xs text-foreground placeholder-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring";
const labelClass = "block text-xs uppercase tracking-wider font-mono text-muted-foreground mb-1.5";

const toList = (text: string) =>
  text
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

/**
 * Edits where the scanner looks and how folder names map to categories.
 */
export const ScannerSettings: React.FC<ScannerSettingsProps> = ({ config, categories, onSaved }) => {
  const { t, categoryName } = useI18n();
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
      setMessage({ ok: true, text: t.scanner.saved });
      onSaved(saved);
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : t.scanner.saveFailed });
    } finally {
      setSaving(false);
    }
  };

  const tree = buildCategoryTree(categories);

  return (
    <div className="space-y-5">
      {/* Library roots */}
      <div>
        <span className={labelClass}>{t.scanner.roots}</span>
        {roots.length === 0 && (
          <p className="text-xs text-amber-700 dark:text-amber-300 mb-2">{t.scanner.rootsHint}</p>
        )}
        <ul className="space-y-1 mb-2">
          {roots.map((r) => (
            <li
              key={r}
              className="flex items-center gap-2 rounded-lg bg-background/80 border border-border px-3 py-1.5"
            >
              <span className="flex-1 font-mono text-xs text-foreground break-all"><Folder className="mr-1.5 inline size-3.5" />{r}</span>
              <button
                type="button"
                onClick={() => setRoots(roots.filter((x) => x !== r))}
                className="text-muted-foreground hover:text-rose-600 dark:hover:text-rose-400 text-xs cursor-pointer"
                title={t.common.remove}
              >
                <X className="size-3.5" />
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
            placeholder={t.scanner.rootPlaceholder}
            className={`${inputClass} flex-1 font-mono`}
          />
          <button
            type="button"
            onClick={handleBrowse}
            className="px-3 py-1.5 rounded-lg bg-muted hover:bg-accent text-foreground text-xs border border-border cursor-pointer"
          >
            <FolderOpen className="size-4" /> {t.scanner.browse}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label className={labelClass}>{t.scanner.archiveDirs}</label>
          <input value={archiveDirs} onChange={(e) => setArchiveDirs(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-xs text-muted-foreground">{t.scanner.archiveHelp}</p>
        </div>
        <div>
          <label className={labelClass}>{t.scanner.ignore}</label>
          <input value={ignore} onChange={(e) => setIgnore(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-xs text-muted-foreground">{t.scanner.ignoreHelp}</p>
        </div>
        <div>
          <label className={labelClass}>{t.scanner.deps}</label>
          <input value={deps} onChange={(e) => setDeps(e.target.value)} className={`${inputClass} w-full`} />
          <p className="mt-1 text-xs text-muted-foreground">{t.scanner.depsHelp}</p>
        </div>
      </div>

      {/* Folder -> category mapping */}
      <div>
        <span className={labelClass}>{t.scanner.mapping}</span>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-1.5 max-h-72 overflow-y-auto pr-1">
          {mapping.map(([folder, category], idx) => (
            <div key={idx} className="flex items-center gap-1.5">
              <input
                value={folder}
                onChange={(e) =>
                  setMapping(mapping.map((m, i) => (i === idx ? [e.target.value.toLowerCase(), m[1]] : m)))
                }
                placeholder={t.scanner.folder}
                className={`${inputClass} w-28 font-mono`}
              />
              <span className="text-muted-foreground/70 text-xs">→</span>
              <select
                value={category}
                onChange={(e) => setMapping(mapping.map((m, i) => (i === idx ? [m[0], e.target.value] : m)))}
                className={`${inputClass} flex-1 min-w-0 cursor-pointer`}
              >
                {!categories.some((c) => c.name === category) && <option value={category}>{t.scanner.missingCategory(category)}</option>}
                {tree.map((root) => (
                  <React.Fragment key={root.id}>
                    <option value={root.name}>{categoryName(root.name)}</option>
                    {root.children.map((child) => (
                      <option key={child.id} value={child.name}>
                        &nbsp;&nbsp;└ {categoryName(child.name)}
                      </option>
                    ))}
                  </React.Fragment>
                ))}
              </select>
              <button
                type="button"
                onClick={() => setMapping(mapping.filter((_, i) => i !== idx))}
                className="text-muted-foreground hover:text-rose-600 dark:hover:text-rose-400 text-xs px-1 cursor-pointer"
                title={t.common.remove}
              >
                <X className="size-3.5" />
              </button>
            </div>
          ))}
        </div>
        <button
          type="button"
          onClick={() => setMapping([...mapping, ["", categories[0]?.name ?? "Other"]])}
          className="mt-2 text-xs text-primary hover:text-primary cursor-pointer"
        >
          {t.scanner.addMapping}
        </button>
      </div>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold cursor-pointer disabled:opacity-50"
        >
          {saving ? t.scanner.saving : t.scanner.save}
        </button>
        {message && <span className={`text-xs ${message.ok ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}>{message.text}</span>}
      </div>
    </div>
  );
};
