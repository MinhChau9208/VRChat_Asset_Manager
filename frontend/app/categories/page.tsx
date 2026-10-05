"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Category,
  CategoryNode,
  LibraryStats,
  buildCategoryTree,
  createCategory,
  deleteCategory,
  getCategories,
  getLibraryStats,
  updateCategory,
} from "@/lib/api";
import { Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";

const inputClass =
  "rounded-lg border border-border bg-background/80 px-3 py-1.5 text-xs sm:text-sm text-foreground placeholder-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring";
const iconButtonClass =
  "h-7 w-7 inline-flex items-center justify-center rounded-md text-xs text-muted-foreground hover:text-foreground hover:bg-muted border border-transparent hover:border-border cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed";

export default function CategoriesPage() {
  const { t, categoryName } = useI18n();
  const [categories, setCategories] = useState<Category[]>([]);
  const [stats, setStats] = useState<LibraryStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [reloadToken, setReloadToken] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const [newName, setNewName] = useState("");
  const [newParent, setNewParent] = useState<number | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState("");

  useEffect(() => {
    let cancelled = false;
    Promise.all([getCategories(), getLibraryStats()])
      .then(([cats, libraryStats]) => {
        if (cancelled) return;
        setCategories(cats);
        setStats(libraryStats);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : t.categories.loadFailed);
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [reloadToken, t]);

  const tree = buildCategoryTree(categories);
  const topLevel = tree.map(({ children, ...c }) => ({ ...c, hasChildren: children.length > 0 }));

  // Runs a mutation, then reloads the list; errors are shown inline.
  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      setReloadToken((t) => t + 1);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.somethingWrong);
      return false;
    } finally {
      setBusy(false);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName.trim()) return;
    if (await run(() => createCategory({ name: newName.trim(), parent_id: newParent }))) {
      setNewName("");
    }
  };

  const handleRename = async (cat: Category) => {
    if (!editName.trim() || editName.trim() === cat.name) {
      setEditingId(null);
      return;
    }
    if (await run(() => updateCategory(cat.id, { name: editName.trim(), parent_id: cat.parent_id }))) {
      setEditingId(null);
    }
  };

  // Moves a category among its siblings and renumbers them 10, 20, 30…
  const handleMove = (siblings: Category[], index: number, delta: number) => {
    const target = index + delta;
    if (target < 0 || target >= siblings.length) return;
    const reordered = [...siblings];
    [reordered[index], reordered[target]] = [reordered[target], reordered[index]];
    run(async () => {
      for (const [i, c] of reordered.entries()) {
        const sortOrder = (i + 1) * 10;
        if (c.sort_order !== sortOrder) {
          await updateCategory(c.id, { name: c.name, parent_id: c.parent_id, sort_order: sortOrder });
        }
      }
    });
  };

  const handleReparent = (cat: Category, parentId: number | null) => {
    run(() => updateCategory(cat.id, { name: cat.name, parent_id: parentId }));
  };

  const handleDelete = (cat: Category) => {
    const count = stats?.by_category[cat.id] ?? 0;
    const name = categoryName(cat.name);
    const message = count > 0 ? t.categories.deleteWithAssets(name, count) : t.categories.deleteConfirm(name);
    if (window.confirm(message)) {
      run(() => deleteCategory(cat.id));
    }
  };

  const renderRow = (cat: Category, siblings: Category[], index: number, node?: CategoryNode) => {
    const isChild = cat.parent_id !== null;
    const count = stats?.by_category[cat.id] ?? 0;
    return (
      <li
        key={cat.id}
        className={`flex flex-wrap items-center gap-2 py-2 px-3 rounded-lg hover:bg-card/70 ${
          isChild ? "ml-6 sm:ml-10" : ""
        }`}
      >
        <div className="flex flex-col">
          <button
            type="button"
            className={iconButtonClass}
            onClick={() => handleMove(siblings, index, -1)}
            disabled={busy || index === 0}
            title={t.categories.moveUp}
          >
            ▲
          </button>
          <button
            type="button"
            className={iconButtonClass}
            onClick={() => handleMove(siblings, index, 1)}
            disabled={busy || index === siblings.length - 1}
            title={t.categories.moveDown}
          >
            ▼
          </button>
        </div>

        <div className="flex-1 min-w-[10rem]">
          {editingId === cat.id ? (
            <input
              autoFocus
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              onBlur={() => handleRename(cat)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleRename(cat);
                if (e.key === "Escape") setEditingId(null);
              }}
              className={`${inputClass} w-full`}
            />
          ) : (
            <button
              type="button"
              onClick={() => {
                setEditingId(cat.id);
                setEditName(cat.name);
              }}
              className={`text-left cursor-text ${isChild ? "text-sm text-foreground" : "text-sm font-semibold text-foreground"}`}
              title={t.categories.rename}
            >
              {categoryName(cat.name)}
              {categoryName(cat.name) !== cat.name && (
                <span className="ml-1.5 text-xs font-normal text-muted-foreground">({cat.name})</span>
              )}
            </button>
          )}
          <div className="text-xs text-muted-foreground">
            {t.common.assets(count)}
            {node && node.children.length > 0 && t.categories.subcategories(node.children.length)}
          </div>
        </div>

        {/* A category with subcategories must stay top-level. */}
        {!(node && node.children.length > 0) && (
          <select
            value={cat.parent_id ?? ""}
            onChange={(e) => handleReparent(cat, e.target.value === "" ? null : Number(e.target.value))}
            disabled={busy}
            className={`${inputClass} w-40 cursor-pointer`}
            title={t.categories.parent}
          >
            <option value="">{t.categories.topLevel}</option>
            {topLevel
              .filter((p) => p.id !== cat.id)
              .map((p) => (
                <option key={p.id} value={p.id}>
                  {t.categories.under(categoryName(p.name))}
                </option>
              ))}
          </select>
        )}

        <button
          type="button"
          onClick={() => handleDelete(cat)}
          disabled={busy || Boolean(node && node.children.length > 0)}
          className={`${iconButtonClass} hover:text-rose-600 dark:hover:text-rose-400`}
          title={node && node.children.length > 0 ? t.categories.deleteChildrenFirst : t.common.delete}
        >
          <Trash2 className="size-4" />
        </button>
      </li>
    );
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      <div className="max-w-3xl mx-auto px-4 py-8 space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <Link href="/" className="text-xs text-muted-foreground hover:text-primary transition-colors">
              {t.common.backToLibrary}
            </Link>
            <h1 className="mt-1 text-xl font-bold tracking-tight text-foreground">{t.categories.title}</h1>
            <p className="text-xs text-muted-foreground mt-0.5">
              {t.categories.intro}
            </p>
          </div>
        </div>

        <form
          onSubmit={handleCreate}
          className="flex flex-col sm:flex-row gap-2 p-4 rounded-xl bg-card/60 border border-border"
        >
          <input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={t.categories.newPlaceholder}
            className={`${inputClass} flex-1`}
          />
          <select
            value={newParent ?? ""}
            onChange={(e) => setNewParent(e.target.value === "" ? null : Number(e.target.value))}
            className={`${inputClass} sm:w-48 cursor-pointer`}
          >
            <option value="">{t.categories.topLevel}</option>
            {topLevel.map((p) => (
              <option key={p.id} value={p.id}>
                {t.categories.under(categoryName(p.name))}
              </option>
            ))}
          </select>
          <button
            type="submit"
            disabled={busy || !newName.trim()}
            className="px-4 py-1.5 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-semibold cursor-pointer disabled:opacity-50"
          >
            {t.common.add}
          </button>
        </form>

        {error && (
          <div className="p-3 rounded-lg bg-rose-100 dark:bg-rose-950/40 border border-rose-300 dark:border-rose-800/60 text-xs text-rose-700 dark:text-rose-300">{error}</div>
        )}

        {isLoading ? (
          <p className="text-xs text-muted-foreground">{t.categories.loading}</p>
        ) : (
          <ul className="space-y-0.5 rounded-xl bg-card/40 border border-border p-2">
            {tree.map((root, i) => (
              <React.Fragment key={root.id}>
                {renderRow(root, tree, i, root)}
                {root.children.map((child, j) => renderRow(child, root.children, j))}
              </React.Fragment>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
