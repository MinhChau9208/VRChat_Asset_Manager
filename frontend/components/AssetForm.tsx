"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  Asset,
  Category,
  CompatAvatar,
  Tag,
  CreateAssetInput,
  UpdateAssetInput,
  getCategories,
  getTags,
  createAsset,
  updateAsset,
  pickFolder,
  buildCategoryTree,
  BoothSuggestion,
  lookupBooth,
  setPreviewFromBooth,
  uploadAssetPreview,
} from "@/lib/api";
import { PreviewDropzone } from "./PreviewDropzone";
import { assetPreviewSrc } from "./AssetCard";
import { CompatibleAvatarsInput } from "./CompatibleAvatarsInput";
import { BoothImportPanel, BoothSelection } from "./BoothImportPanel";
import { TriangleAlert, X } from "lucide-react";

interface AssetFormProps {
  initialData?: Asset;
  mode: "create" | "edit";
  onSubmitSuccess: (asset: Asset) => void;
  onCancel?: () => void;
}

export const AssetForm: React.FC<AssetFormProps> = ({
  initialData,
  mode,
  onSubmitSuccess,
  onCancel,
}) => {
  // Form fields
  const [name, setName] = useState(initialData?.name || "");
  const [categoryId, setCategoryId] = useState<number | null>(
    initialData?.category_id ?? null
  );
  const [author, setAuthor] = useState(initialData?.author || "");
  const [boothUrl, setBoothUrl] = useState(initialData?.booth_url || "");
  const [localPath, setLocalPath] = useState(initialData?.local_path || "");
  const [description, setDescription] = useState(initialData?.description || "");
  const [tags, setTags] = useState<string[]>(initialData?.tags || []);
  const [compatibleAvatars, setCompatibleAvatars] = useState<CompatAvatar[]>(
    initialData?.compatible_avatars || []
  );

  // Tag input state
  const [tagInput, setTagInput] = useState("");

  // BOOTH import state; the chosen image is downloaded after the asset is saved.
  const [boothSuggestion, setBoothSuggestion] = useState<BoothSuggestion | null>(null);
  const [isFetchingBooth, setIsFetchingBooth] = useState(false);
  const [boothError, setBoothError] = useState<string | null>(null);
  const [pendingPreviewUrl, setPendingPreviewUrl] = useState<string | null>(null);
  // A picked / dropped / pasted image, uploaded after the asset is saved.
  const [pendingPreviewFile, setPendingPreviewFile] = useState<{ file: File; src: string } | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const choosePreviewFile = (file: File) => {
    setPendingPreviewFile((prev) => {
      if (prev) URL.revokeObjectURL(prev.src);
      return { file, src: URL.createObjectURL(file) };
    });
    setPendingPreviewUrl(null);
    setPreviewError(null);
  };

  // Data loading states
  const [categories, setCategories] = useState<Category[]>([]);
  const [existingTags, setExistingTags] = useState<Tag[]>([]);
  const [isLoadingMetadata, setIsLoadingMetadata] = useState(true);

  // Submission & validation states
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isPickingFolder, setIsPickingFolder] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<{
    name?: string;
    boothUrl?: string;
    tagInput?: string;
  }>({});

  // Load categories and existing tags on mount
  useEffect(() => {
    let isMounted = true;

    async function loadMetadata() {
      setIsLoadingMetadata(true);
      try {
        const [cats, allTags] = await Promise.all([
          getCategories().catch(() => [] as Category[]),
          getTags().catch(() => [] as Tag[]),
        ]);
        if (isMounted) {
          setCategories(cats);
          setExistingTags(allTags);
        }
      } finally {
        if (isMounted) {
          setIsLoadingMetadata(false);
        }
      }
    }

    loadMetadata();

    return () => {
      isMounted = false;
    };
  }, []);

  const handleFetchBooth = async () => {
    setIsFetchingBooth(true);
    setBoothError(null);
    try {
      setBoothSuggestion(await lookupBooth(boothUrl.trim()));
    } catch (err) {
      setBoothError(err instanceof Error ? err.message : "BOOTH lookup failed");
    } finally {
      setIsFetchingBooth(false);
    }
  };

  const handleApplyBooth = (sel: BoothSelection) => {
    if (boothSuggestion) setBoothUrl(boothSuggestion.booth_url);
    if (sel.name) setName(sel.name);
    if (sel.author) setAuthor(sel.author);
    if (sel.categoryId !== undefined) setCategoryId(sel.categoryId);
    if (sel.tags.length > 0) {
      setTags((prev) => [...prev, ...sel.tags.filter((t) => !prev.some((p) => p.toLowerCase() === t.toLowerCase()))]);
    }
    setCompatibleAvatars((prev) => [
      ...prev,
      ...sel.compatibleAvatars.filter(
        (c) => !prev.some((p) => p.avatar_name.toLowerCase() === c.avatar_name.toLowerCase())
      ),
    ]);
    if (sel.imageUrl) {
      setPendingPreviewUrl(sel.imageUrl);
      setPendingPreviewFile(null);
    }
    setBoothSuggestion(null);
  };

  // Immediate validation check
  const validate = (): boolean => {
    const errors: { name?: string; boothUrl?: string } = {};

    if (!name.trim()) {
      errors.name = "Asset name is required.";
    }

    if (boothUrl.trim()) {
      try {
        const parsed = new URL(boothUrl.trim());
        if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
          errors.boothUrl = "BOOTH URL must start with http:// or https://";
        }
      } catch {
        errors.boothUrl = "Please enter a valid URL (e.g., https://booth.pm/...)";
      }
    }

    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  // Add tag handler
  const handleAddTag = useCallback(
    (tagToAdd?: string) => {
      const candidate = (tagToAdd !== undefined ? tagToAdd : tagInput).trim();
      if (!candidate) return;

      // Case-insensitive duplicate check
      const exists = tags.some(
        (t) => t.toLowerCase() === candidate.toLowerCase()
      );
      if (exists) {
        setFieldErrors((prev) => ({
          ...prev,
          tagInput: `Tag "${candidate}" has already been added.`,
        }));
        return;
      }

      setTags((prev) => [...prev, candidate]);
      setTagInput("");
      setFieldErrors((prev) => ({ ...prev, tagInput: undefined }));
    },
    [tagInput, tags]
  );

  // Remove tag handler
  const handleRemoveTag = (indexToRemove: number) => {
    setTags((prev) => prev.filter((_, idx) => idx !== indexToRemove));
  };

  // Pick folder button handler
  const handlePickFolder = async () => {
    setIsPickingFolder(true);
    try {
      const selected = await pickFolder();
      if (selected) {
        setLocalPath(selected);
      }
    } catch {
      // Ignored - manual entry is always available
    } finally {
      setIsPickingFolder(false);
    }
  };

  // Form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    // If user has unsubmitted text in tag input, add it if valid
    if (tagInput.trim()) {
      const candidate = tagInput.trim();
      if (!tags.some((t) => t.toLowerCase() === candidate.toLowerCase())) {
        tags.push(candidate);
      }
      setTagInput("");
    }

    if (!validate()) {
      return;
    }

    setIsSubmitting(true);

    try {
      const payload: CreateAssetInput | UpdateAssetInput = {
        name: name.trim(),
        category_id: categoryId,
        author: author.trim(),
        booth_url: boothUrl.trim(),
        local_path: localPath.trim(),
        description: description.trim(),
        tags: tags,
        compatible_avatars: compatibleAvatars,
      };

      let result: Asset;
      if (mode === "create") {
        result = await createAsset(payload);
      } else {
        if (!initialData?.id) {
          throw new Error("Missing asset ID for update");
        }
        result = await updateAsset(initialData.id, payload);
      }

      // The asset is saved at this point; a failed preview does not undo that.
      try {
        if (pendingPreviewFile) {
          result = await uploadAssetPreview(result.id, pendingPreviewFile.file);
        } else if (pendingPreviewUrl) {
          result = await setPreviewFromBooth(result.id, pendingPreviewUrl);
        }
      } catch (previewErr) {
        console.warn("Failed to save preview:", previewErr);
      }

      onSubmitSuccess(result);
    } catch (err: unknown) {
      const message =
        err instanceof Error
          ? err.message
          : "An unexpected error occurred while saving.";
      setFormError(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Filter available existing tags to suggest
  const suggestedTags = existingTags.filter(
    (existing) =>
      !tags.some((t) => t.toLowerCase() === existing.name.toLowerCase())
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Global Form Error Banner */}
      {formError && (
        <div className="p-3.5 rounded-xl text-xs bg-rose-950/50 border border-rose-800/70 text-rose-300 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <TriangleAlert className="size-4 shrink-0" />
            <span>{formError}</span>
          </div>
          <button
            type="button"
            onClick={() => setFormError(null)}
            className="text-neutral-400 hover:text-white cursor-pointer px-1"
          >
            <X className="size-4" />
          </button>
        </div>
      )}

      {/* Preview */}
      <div className="flex flex-col gap-4 rounded-xl border border-border bg-card/40 p-5 sm:flex-row sm:items-start">
        <div className="w-full max-w-[200px] shrink-0">
          <PreviewDropzone
            src={pendingPreviewFile?.src ?? pendingPreviewUrl ?? (initialData ? assetPreviewSrc(initialData) : null)}
            onFile={choosePreviewFile}
            onError={setPreviewError}
            pasteAnywhere
          />
        </div>
        <div className="space-y-1.5 text-sm text-muted-foreground">
          <h3 className="text-sm font-medium text-foreground">Preview image</h3>
          <p>Click the square, drop an image on it, or paste one with Ctrl+V (a screenshot of the BOOTH page works).</p>
          <p>
            You can also pick one of the BOOTH images with <span className="text-foreground">Fetch from BOOTH</span> below.
            The image is saved when you save the asset.
          </p>
          {(pendingPreviewFile || pendingPreviewUrl) && (
            <p className="text-emerald-400">New preview selected — it will be saved with the asset.</p>
          )}
          {previewError && <p className="text-destructive">{previewError}</p>}
        </div>
      </div>

      {/* Basic Info Section */}
      <div className="rounded-xl border border-neutral-800 bg-neutral-900/60 p-5 space-y-4 backdrop-blur-sm">
        <h3 className="text-xs uppercase font-mono tracking-wider text-cyan-400 font-semibold">
          General Information
        </h3>

        {/* Asset Name */}
        <div>
          <label
            htmlFor="asset-name"
            className="block text-xs font-medium text-neutral-300 mb-1"
          >
            Asset Name <span className="text-rose-400">*</span>
          </label>
          <input
            id="asset-name"
            type="text"
            required
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (fieldErrors.name) {
                setFieldErrors((prev) => ({ ...prev, name: undefined }));
              }
            }}
            placeholder="e.g. Cute Anime Hair, Gothic Dress, Selestia..."
            className={`w-full rounded-lg border bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 focus:outline-none focus:ring-1 transition-all ${
              fieldErrors.name
                ? "border-rose-500 focus:border-rose-500 focus:ring-rose-500"
                : "border-neutral-800 focus:border-cyan-500 focus:ring-cyan-500"
            }`}
          />
          {fieldErrors.name && (
            <p className="mt-1 text-xs text-rose-400">{fieldErrors.name}</p>
          )}
        </div>

        {/* Category & Author Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Category Dropdown */}
          <div>
            <label
              htmlFor="asset-category"
              className="block text-xs font-medium text-neutral-300 mb-1"
            >
              Category
            </label>
            <select
              id="asset-category"
              value={categoryId ?? ""}
              onChange={(e) => {
                const val = e.target.value;
                setCategoryId(val === "" ? null : Number(val));
              }}
              disabled={isLoadingMetadata}
              className="w-full rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all cursor-pointer disabled:opacity-50"
            >
              <option value="">None / Uncategorized</option>
              {buildCategoryTree(categories).map((root) =>
                root.children.length === 0 ? (
                  <option key={root.id} value={root.id}>
                    {root.name}
                  </option>
                ) : (
                  <optgroup key={root.id} label={root.name}>
                    <option value={root.id}>{root.name} (general)</option>
                    {root.children.map((child) => (
                      <option key={child.id} value={child.id}>
                        {child.name}
                      </option>
                    ))}
                  </optgroup>
                )
              )}
            </select>
            {isLoadingMetadata && (
              <p className="mt-1 text-xs text-neutral-500">
                Loading categories...
              </p>
            )}
          </div>

          {/* Author */}
          <div>
            <label
              htmlFor="asset-author"
              className="block text-xs font-medium text-neutral-300 mb-1"
            >
              Author / Creator
            </label>
            <input
              id="asset-author"
              type="text"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
              placeholder="e.g. Komado, 久, Booth Creator..."
              className="w-full rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all"
            />
          </div>
        </div>

        {/* BOOTH URL */}
        <div>
          <label
            htmlFor="asset-booth-url"
            className="block text-xs font-medium text-neutral-300 mb-1"
          >
            BOOTH URL
          </label>
          <div className="flex gap-2">
          <input
            id="asset-booth-url"
            type="url"
            value={boothUrl}
            onChange={(e) => {
              setBoothUrl(e.target.value);
              if (fieldErrors.boothUrl) {
                setFieldErrors((prev) => ({ ...prev, boothUrl: undefined }));
              }
            }}
            placeholder="https://booth.pm/en/items/..."
            className={`w-full rounded-lg border bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 focus:outline-none focus:ring-1 transition-all ${
              fieldErrors.boothUrl
                ? "border-rose-500 focus:border-rose-500 focus:ring-rose-500"
                : "border-neutral-800 focus:border-cyan-500 focus:ring-cyan-500"
            }`}
          />
          <button
            type="button"
            onClick={handleFetchBooth}
            disabled={!boothUrl.trim() || isFetchingBooth}
            className="px-3 py-2 rounded-lg bg-red-800/80 hover:bg-red-700 text-white text-xs font-semibold border border-red-700/60 transition-colors cursor-pointer shrink-0 disabled:opacity-40 disabled:cursor-not-allowed"
            title="Fill name, author, category, preview and compatible avatars from BOOTH"
          >
            {isFetchingBooth ? "Fetching…" : "Fetch from BOOTH"}
          </button>
          </div>
          {fieldErrors.boothUrl && (
            <p className="mt-1 text-xs text-rose-400">{fieldErrors.boothUrl}</p>
          )}
          {boothError && <p className="mt-1 text-xs text-rose-400">{boothError}</p>}
          {boothSuggestion && (
            <BoothImportPanel
              suggestion={boothSuggestion}
              onApply={handleApplyBooth}
              onClose={() => setBoothSuggestion(null)}
            />
          )}
        </div>

        {/* Description */}
        <div>
          <label
            htmlFor="asset-description"
            className="block text-xs font-medium text-neutral-300 mb-1"
          >
            Description & Notes
          </label>
          <textarea
            id="asset-description"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Add notes, installation requirements, shader details, or instructions..."
            className="w-full rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all resize-y"
          />
        </div>
      </div>

      {/* Filesystem Location Section */}
      <div className="rounded-xl border border-neutral-800 bg-neutral-900/60 p-5 space-y-3 backdrop-blur-sm">
        <h3 className="text-xs uppercase font-mono tracking-wider text-cyan-400 font-semibold flex items-center justify-between">
          <span>Local Filesystem Path</span>
          <span className="text-xs text-neutral-400 lowercase font-normal">
            safe reference &bull; files are never deleted
          </span>
        </h3>

        <div>
          <label
            htmlFor="asset-local-path"
            className="block text-xs font-medium text-neutral-300 mb-1"
          >
            Windows Directory or File Path
          </label>
          <div className="flex gap-2">
            <input
              id="asset-local-path"
              type="text"
              value={localPath}
              onChange={(e) => setLocalPath(e.target.value)}
              placeholder="e.g. C:\VRChat Assets\Hair\CuteHair or \\server\share\..."
              className="flex-1 font-mono rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-2 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all"
            />
            <button
              type="button"
              onClick={handlePickFolder}
              disabled={isPickingFolder}
              title="Open Windows Folder Browser"
              className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg bg-neutral-800 hover:bg-neutral-700 active:bg-neutral-800 text-neutral-200 hover:text-white border border-neutral-700 text-xs font-medium transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed shrink-0"
            >
              {isPickingFolder ? (
                <>
                  <div className="h-3 w-3 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                  <span className="hidden sm:inline">Browsing...</span>
                </>
              ) : (
                <>
                  <svg
                    className="w-3.5 h-3.5 text-cyan-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                    />
                  </svg>
                  <span>Browse...</span>
                </>
              )}
            </button>
          </div>
          <p className="mt-1 text-xs text-neutral-500">
            Paste any local Windows folder path or UNC network share. Deleting an
            asset record will never delete or modify the contents of this path.
          </p>
        </div>
      </div>

      {/* Tags Section */}
      <div className="rounded-xl border border-neutral-800 bg-neutral-900/60 p-5 space-y-4 backdrop-blur-sm">
        <h3 className="text-xs uppercase font-mono tracking-wider text-cyan-400 font-semibold">
          Tags & Metadata
        </h3>

        {/* Assigned Tag Chips */}
        <div>
          <label className="block text-xs font-medium text-neutral-300 mb-2">
            Assigned Tags ({tags.length})
          </label>
          {tags.length === 0 ? (
            <p className="text-xs text-neutral-500 italic">No tags assigned yet.</p>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {tags.map((tag, idx) => (
                <span
                  key={`${tag}-${idx}`}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-neutral-800 text-neutral-200 border border-neutral-700/80 group transition-all"
                >
                  <span className="text-cyan-400/80 text-xs">#</span>
                  <span>{tag}</span>
                  <button
                    type="button"
                    onClick={() => handleRemoveTag(idx)}
                    className="text-neutral-400 hover:text-rose-400 transition-colors cursor-pointer p-0.5"
                    title={`Remove tag "${tag}"`}
                  >
                    <X className="size-3.5" />
                  </button>
                </span>
              ))}
            </div>
          )}
        </div>

        {/* Tag Input Field */}
        <div>
          <div className="flex gap-2">
            <input
              type="text"
              value={tagInput}
              onChange={(e) => {
                setTagInput(e.target.value);
                if (fieldErrors.tagInput) {
                  setFieldErrors((prev) => ({ ...prev, tagInput: undefined }));
                }
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === ",") {
                  e.preventDefault();
                  handleAddTag();
                }
              }}
              placeholder="Type tag name and press Enter or comma..."
              className="flex-1 rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-1.5 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all"
            />
            <button
              type="button"
              onClick={() => handleAddTag()}
              className="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium border border-neutral-700 transition-colors cursor-pointer shrink-0"
            >
              + Add
            </button>
          </div>
          {fieldErrors.tagInput && (
            <p className="mt-1 text-xs text-rose-400">{fieldErrors.tagInput}</p>
          )}
        </div>

        {/* Quick-Add Suggestions from Existing Tags */}
        {suggestedTags.length > 0 && (
          <div>
            <span className="block text-xs text-neutral-400 mb-1.5">
              Suggestions from your library (click to add):
            </span>
            <div className="flex flex-wrap gap-1 max-h-24 overflow-y-auto pr-1">
              {suggestedTags.slice(0, 20).map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => handleAddTag(t.name)}
                  className="px-2 py-0.5 rounded text-xs bg-neutral-950 hover:bg-neutral-800 text-neutral-400 hover:text-cyan-300 border border-neutral-800 hover:border-neutral-700 transition-colors cursor-pointer"
                >
                  +{t.name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Compatible Avatars */}
        <div className="pt-2 border-t border-neutral-800/60">
          <CompatibleAvatarsInput
            value={compatibleAvatars}
            onChange={setCompatibleAvatars}
            selfId={initialData?.id}
          />
        </div>
      </div>

      {/* Form Action Buttons */}
      <div className="flex items-center justify-end gap-3 pt-2">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-neutral-400 hover:text-neutral-200 bg-neutral-900 hover:bg-neutral-800 border border-neutral-800 transition-colors cursor-pointer disabled:opacity-50"
          >
            Cancel
          </button>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="inline-flex items-center justify-center gap-2 px-6 py-2.5 rounded-xl bg-cyan-600 hover:bg-cyan-500 active:bg-cyan-600 text-white text-xs font-semibold shadow-lg shadow-cyan-950/40 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSubmitting ? (
            <>
              <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
              <span>{mode === "create" ? "Creating..." : "Saving..."}</span>
            </>
          ) : (
            <span>{mode === "create" ? "Create Asset" : "Save Changes"}</span>
          )}
        </button>
      </div>
    </form>
  );
};
