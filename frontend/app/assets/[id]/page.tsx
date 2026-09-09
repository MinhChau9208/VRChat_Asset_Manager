"use client";

import React, { useEffect, useState, useCallback, useRef } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  getAssetByID,
  getAssetStatus,
  openAssetFolder,
  getAssetPreviewUrl,
  uploadAssetPreview,
  deleteAssetPreview,
  deleteAsset,
  toggleAssetFavorite,
  Asset,
  AssetStatus,
} from "@/lib/api";
import { AssetForm } from "@/components/AssetForm";
import { DeleteConfirmationModal } from "@/components/DeleteConfirmationModal";

export default function AssetDetailPage() {
  const params = useParams();
  const router = useRouter();
  const id = params?.id as string;

  const [asset, setAsset] = useState<Asset | null>(null);
  const [fileStatus, setFileStatus] = useState<AssetStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCheckingStatus, setIsCheckingStatus] = useState(false);
  const [isNotFound, setIsNotFound] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Edit & Delete modal states
  const [isEditing, setIsEditing] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  // Preview upload & delete states
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isUploadingPreview, setIsUploadingPreview] = useState(false);
  const [isDeletingPreview, setIsDeletingPreview] = useState(false);
  const [previewImageError, setPreviewImageError] = useState(false);

  // Open folder state
  const [isOpening, setIsOpening] = useState(false);
  const [actionFeedback, setActionFeedback] = useState<{
    type: "success" | "error";
    message: string;
  } | null>(null);

  const loadData = useCallback(async () => {
    if (!id) return;

    setIsLoading(true);
    setIsNotFound(false);
    setErrorMessage(null);
    setActionFeedback(null);
    setFileStatus(null);
    setPreviewImageError(false);

    try {
      const assetData = await getAssetByID(id);
      setAsset(assetData);
      setIsLoading(false);

      // If asset has a local path, check file existence status
      if (assetData.local_path && assetData.local_path.trim() !== "") {
        setIsCheckingStatus(true);
        try {
          const status = await getAssetStatus(id);
          setFileStatus(status);
        } catch (statusErr) {
          console.warn("Failed to check asset file status:", statusErr);
          setFileStatus({ exists: false });
        } finally {
          setIsCheckingStatus(false);
        }
      }
    } catch (err: unknown) {
      setIsLoading(false);
      const message = err instanceof Error ? err.message : "Failed to load asset";
      if (message.toLowerCase().includes("not found")) {
        setIsNotFound(true);
      } else {
        setErrorMessage(message);
      }
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSelectFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !asset) return;

    // Validate type
    const validTypes = ["image/jpeg", "image/png", "image/webp"];
    if (!validTypes.includes(file.type)) {
      setActionFeedback({
        type: "error",
        message: "Invalid file type. Please select a JPEG, PNG, or WebP image.",
      });
      if (fileInputRef.current) fileInputRef.current.value = "";
      return;
    }

    // Validate size (10MB)
    if (file.size > 10 * 1024 * 1024) {
      setActionFeedback({
        type: "error",
        message: "File size exceeds the 10MB limit.",
      });
      if (fileInputRef.current) fileInputRef.current.value = "";
      return;
    }

    setIsUploadingPreview(true);
    setActionFeedback(null);

    try {
      const updated = await uploadAssetPreview(asset.id, file);
      setAsset(updated);
      setPreviewImageError(false);
      setActionFeedback({
        type: "success",
        message: "Preview image updated successfully.",
      });
      setTimeout(() => {
        setActionFeedback((prev) => (prev?.type === "success" ? null : prev));
      }, 4000);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to upload preview";
      setActionFeedback({
        type: "error",
        message: msg,
      });
    } finally {
      setIsUploadingPreview(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const handleDeletePreview = async () => {
    if (!asset || isDeletingPreview) return;

    setIsDeletingPreview(true);
    setActionFeedback(null);

    try {
      await deleteAssetPreview(asset.id);
      setAsset((prev) => (prev ? { ...prev, preview_path: "" } : null));
      setPreviewImageError(false);
      setActionFeedback({
        type: "success",
        message: "Preview image deleted.",
      });
      setTimeout(() => {
        setActionFeedback((prev) => (prev?.type === "success" ? null : prev));
      }, 4000);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to delete preview";
      setActionFeedback({
        type: "error",
        message: msg,
      });
    } finally {
      setIsDeletingPreview(false);
    }
  };

  const handleOpenFolder = async () => {
    if (!asset || !asset.local_path || isOpening) return;

    setIsOpening(true);
    setActionFeedback(null);

    try {
      await openAssetFolder(asset.id);
      setActionFeedback({
        type: "success",
        message: "Opened in File Explorer.",
      });
      setTimeout(() => {
        setActionFeedback((prev) => (prev?.type === "success" ? null : prev));
      }, 4000);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Failed to open folder";
      setActionFeedback({
        type: "error",
        message,
      });
    } finally {
      setIsOpening(false);
    }
  };

  const handleEditSuccess = (updated: Asset) => {
    setAsset(updated);
    setIsEditing(false);
    setActionFeedback({
      type: "success",
      message: "Asset details updated successfully.",
    });
    setTimeout(() => {
      setActionFeedback((prev) => (prev?.type === "success" ? null : prev));
    }, 4000);

    if (updated.local_path && updated.local_path.trim() !== "") {
      setIsCheckingStatus(true);
      getAssetStatus(updated.id)
        .then((s) => setFileStatus(s))
        .catch(() => setFileStatus({ exists: false }))
        .finally(() => setIsCheckingStatus(false));
    } else {
      setFileStatus(null);
    }
  };

  const handleToggleFavorite = async () => {
    if (!asset) return;
    const nextVal = !asset.is_favorite;
    setAsset((prev) => (prev ? { ...prev, is_favorite: nextVal } : null));

    try {
      await toggleAssetFavorite(asset.id, nextVal);
      setActionFeedback({
        type: "success",
        message: nextVal ? "Added to favorites." : "Removed from favorites.",
      });
      setTimeout(() => {
        setActionFeedback((prev) => (prev?.type === "success" ? null : prev));
      }, 3000);
    } catch (err: unknown) {
      console.error("Failed to toggle favorite:", err);
      setAsset((prev) => (prev ? { ...prev, is_favorite: !nextVal } : null));
      setActionFeedback({
        type: "error",
        message: "Failed to update favorite status.",
      });
    }
  };

  const handleDeleteConfirm = async () => {
    if (!id) return;
    setIsDeleting(true);
    try {
      await deleteAsset(id);
      setIsDeleteModalOpen(false);
      router.push("/");
    } catch (err: unknown) {
      setIsDeleting(false);
      setIsDeleteModalOpen(false);
      const message =
        err instanceof Error ? err.message : "Failed to delete asset";
      setActionFeedback({ type: "error", message });
    }
  };

  // 1. Loading State Skeleton
  if (isLoading) {
    return (
      <main className="min-h-screen bg-neutral-950 text-neutral-100 p-6 sm:p-8 flex flex-col items-center">
        <div className="w-full max-w-4xl space-y-6">
          {/* Breadcrumb Skeleton */}
          <div className="h-5 w-36 bg-neutral-900 rounded animate-pulse" />

          {/* Main Card Skeleton */}
          <div className="rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 md:p-8 backdrop-blur-md">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
              <div className="aspect-[4/3] rounded-xl bg-neutral-800/60 animate-pulse" />
              <div className="space-y-4">
                <div className="h-6 w-24 bg-neutral-800/80 rounded-full animate-pulse" />
                <div className="h-9 w-3/4 bg-neutral-800 rounded animate-pulse" />
                <div className="h-4 w-1/3 bg-neutral-800/70 rounded animate-pulse" />
                <div className="h-20 w-full bg-neutral-800/40 rounded-lg animate-pulse" />
                <div className="flex gap-2 pt-4">
                  <div className="h-6 w-16 bg-neutral-800 rounded-md animate-pulse" />
                  <div className="h-6 w-20 bg-neutral-800 rounded-md animate-pulse" />
                </div>
                <div className="h-12 w-full bg-neutral-800/50 rounded-xl mt-6 animate-pulse" />
              </div>
            </div>
          </div>
        </div>
      </main>
    );
  }

  // 2. 404 Not Found State
  if (isNotFound) {
    return (
      <main className="min-h-screen bg-neutral-950 text-neutral-100 p-6 flex flex-col items-center justify-center">
        <div className="rounded-2xl border border-neutral-800 bg-neutral-900/80 p-8 sm:p-10 text-center max-w-md w-full shadow-2xl backdrop-blur-md">
          <div className="w-14 h-14 mx-auto mb-4 rounded-2xl bg-neutral-800/80 border border-neutral-700 flex items-center justify-center text-2xl text-neutral-400">
            🔍
          </div>
          <h1 className="text-xl font-bold text-white mb-2">Asset not found</h1>
          <p className="text-xs text-neutral-400 mb-6 leading-relaxed">
            The requested asset (#{id}) does not exist in your library or may have been deleted.
          </p>
          <Link
            href="/"
            className="inline-flex items-center justify-center gap-2 w-full rounded-xl bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold py-2.5 px-4 transition-colors shadow-lg shadow-cyan-950/40"
          >
            ← Back to Asset Library
          </Link>
        </div>
      </main>
    );
  }

  // 3. Network / Server Error State
  if (errorMessage || !asset) {
    return (
      <main className="min-h-screen bg-neutral-950 text-neutral-100 p-6 flex flex-col items-center justify-center">
        <div className="rounded-2xl border border-rose-900/40 bg-rose-950/20 p-8 sm:p-10 text-center max-w-md w-full shadow-2xl backdrop-blur-md">
          <div className="w-14 h-14 mx-auto mb-4 rounded-2xl bg-rose-900/30 border border-rose-800/50 flex items-center justify-center text-2xl text-rose-300">
            ⚠️
          </div>
          <h1 className="text-xl font-bold text-rose-300 mb-2">Unable to load asset</h1>
          <p className="text-xs text-neutral-400 mb-6 leading-relaxed">
            {errorMessage || "An unexpected error occurred while connecting to the backend server."}
          </p>
          <div className="flex gap-3">
            <button
              onClick={loadData}
              className="flex-1 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium py-2.5 px-4 transition-colors border border-neutral-700 cursor-pointer"
            >
              Retry
            </button>
            <Link
              href="/"
              className="flex-1 inline-flex items-center justify-center rounded-xl bg-neutral-900 hover:bg-neutral-800 text-neutral-400 hover:text-neutral-200 text-xs font-medium py-2.5 px-4 transition-colors border border-neutral-800"
            >
              Back to Library
            </Link>
          </div>
        </div>
      </main>
    );
  }

  const hasLocalPath = Boolean(asset.local_path && asset.local_path.trim() !== "");
  const previewSrc = asset.preview_path
    ? asset.preview_path.startsWith("http://") || asset.preview_path.startsWith("https://")
      ? asset.preview_path
      : getAssetPreviewUrl(asset.id, asset.updated_at)
    : null;
  const hasPreview = Boolean(previewSrc && !previewImageError);

  return (
    <main className="min-h-screen bg-neutral-950 text-neutral-100 p-4 sm:p-6 md:p-8">
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Navigation / Header Bar */}
        <div className="flex items-center justify-between">
          <Link
            href="/"
            className="inline-flex items-center gap-2 text-xs font-medium text-neutral-400 hover:text-cyan-400 transition-colors group"
          >
            <svg
              className="w-4 h-4 transform transition-transform group-hover:-translate-x-0.5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 19l-7-7m0 0l7-7m-7 7h18"
              />
            </svg>
            Back to Library
          </Link>

          <div className="flex items-center gap-2">
            {!isEditing && (
              <>
                <button
                  type="button"
                  onClick={handleToggleFavorite}
                  id="favorite-asset-btn"
                  className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border text-xs font-semibold transition-all cursor-pointer ${
                    asset.is_favorite
                      ? "bg-rose-950/70 border-rose-500/80 text-rose-300 shadow-sm shadow-rose-950/40"
                      : "bg-neutral-800 hover:bg-neutral-700 text-neutral-300 border-neutral-700 hover:text-white"
                  }`}
                  title={asset.is_favorite ? "Remove from favorites" : "Add to favorites"}
                >
                  <span className={asset.is_favorite ? "text-rose-400 font-bold" : "text-neutral-400 font-bold"}>
                    {asset.is_favorite ? "♥" : "♡"}
                  </span>
                  <span>{asset.is_favorite ? "Favorited" : "Favorite"}</span>
                </button>

                <button
                  onClick={() => setIsEditing(true)}
                  id="edit-asset-btn"
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 active:bg-neutral-800 text-neutral-200 hover:text-white border border-neutral-700 text-xs font-semibold transition-colors cursor-pointer"
                  title="Edit asset details"
                >
                  <svg className="w-3.5 h-3.5 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                  </svg>
                  <span>Edit</span>
                </button>

                <button
                  onClick={() => setIsDeleteModalOpen(true)}
                  id="delete-asset-btn"
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 active:bg-rose-500/30 text-rose-400 hover:text-rose-300 border border-rose-500/30 text-xs font-semibold transition-colors cursor-pointer"
                  title="Delete asset from library"
                >
                  <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  <span>Delete</span>
                </button>
              </>
            )}

            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-neutral-900 border border-neutral-800 text-neutral-500">
              ID: #{asset.id}
            </span>
          </div>
        </div>

        {/* Action Feedback Banner (Success or Error) */}
        {actionFeedback && (
          <div
            className={`p-3.5 rounded-xl text-xs flex items-center justify-between border transition-all ${
              actionFeedback.type === "success"
                ? "bg-emerald-950/40 border-emerald-800/60 text-emerald-300"
                : "bg-rose-950/40 border-rose-800/60 text-rose-300"
            }`}
          >
            <div className="flex items-center gap-2">
              <span>{actionFeedback.type === "success" ? "✓" : "⚠️"}</span>
              <span>{actionFeedback.message}</span>
            </div>
            <button
              onClick={() => setActionFeedback(null)}
              className="text-neutral-400 hover:text-white text-xs px-2 py-0.5 rounded cursor-pointer"
            >
              ✕
            </button>
          </div>
        )}

        {/* Detail Container Card */}
        {isEditing ? (
          <div className="rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 md:p-8 backdrop-blur-md shadow-2xl space-y-6">
            <div className="border-b border-neutral-800/80 pb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-bold text-white tracking-tight flex items-center gap-2">
                  <span className="text-cyan-400">✏️</span>
                  <span>Edit Asset</span>
                </h2>
                <p className="text-xs text-neutral-400 mt-0.5">
                  Update asset metadata, category, booth link, local path, or tags.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setIsEditing(false)}
                className="text-xs text-neutral-400 hover:text-white px-2.5 py-1 rounded-lg bg-neutral-800 hover:bg-neutral-750 border border-neutral-700 cursor-pointer"
              >
                Cancel Edit
              </button>
            </div>

            <AssetForm
              mode="edit"
              initialData={asset}
              onSubmitSuccess={handleEditSuccess}
              onCancel={() => setIsEditing(false)}
            />
          </div>
        ) : (
          <div className="rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 md:p-8 backdrop-blur-md shadow-2xl space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            {/* Left: Preview Section */}
            <div className="space-y-4">
              <div className="rounded-xl border border-neutral-800 bg-neutral-950 aspect-[4/3] flex items-center justify-center overflow-hidden relative group">
                {hasPreview && previewSrc ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={previewSrc}
                    alt={asset.name}
                    onError={() => setPreviewImageError(true)}
                    className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                  />
                ) : (
                  <div className="flex flex-col items-center justify-center p-6 text-center">
                    <div className="w-16 h-16 rounded-2xl bg-neutral-900 border border-neutral-800 flex items-center justify-center text-3xl mb-3 shadow-inner">
                      📦
                    </div>
                    <span className="text-xs uppercase font-mono tracking-wider text-neutral-500">
                      No Preview Available
                    </span>
                    <span className="text-[10px] text-neutral-600 mt-1">
                      Upload a JPEG, PNG, or WebP image
                    </span>
                  </div>
                )}
              </div>

              {/* Preview Management Controls */}
              <div className="space-y-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  className="hidden"
                  onChange={handleSelectFile}
                />

                {!asset.preview_path ? (
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    disabled={isUploadingPreview}
                    className="inline-flex items-center justify-center gap-2 w-full rounded-xl bg-cyan-600/90 hover:bg-cyan-500 active:bg-cyan-600 text-white text-xs font-semibold py-2.5 px-4 transition-colors cursor-pointer shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isUploadingPreview ? (
                      <>
                        <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                        <span>Uploading Preview...</span>
                      </>
                    ) : (
                      <>
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                          />
                        </svg>
                        <span>Upload Preview</span>
                      </>
                    )}
                  </button>
                ) : (
                  <div className="flex gap-2">
                    <button
                      onClick={() => fileInputRef.current?.click()}
                      disabled={isUploadingPreview || isDeletingPreview}
                      className="flex-1 inline-flex items-center justify-center gap-1.5 rounded-xl bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium py-2 px-3 transition-colors border border-neutral-700/80 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {isUploadingPreview ? (
                        <>
                          <div className="h-3 w-3 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                          <span>Replacing...</span>
                        </>
                      ) : (
                        <>
                          <svg className="w-3.5 h-3.5 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                            />
                          </svg>
                          <span>Replace Preview</span>
                        </>
                      )}
                    </button>

                    <button
                      onClick={handleDeletePreview}
                      disabled={isUploadingPreview || isDeletingPreview}
                      className="inline-flex items-center justify-center gap-1.5 rounded-xl bg-neutral-900 hover:bg-rose-950/40 text-neutral-400 hover:text-rose-300 text-xs font-medium py-2 px-3 transition-colors border border-neutral-800 hover:border-rose-900/50 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                      title="Delete preview"
                    >
                      {isDeletingPreview ? (
                        <div className="h-3 w-3 animate-spin rounded-full border-2 border-rose-400 border-t-transparent" />
                      ) : (
                        <svg className="w-3.5 h-3.5 text-rose-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                          />
                        </svg>
                      )}
                      <span>Delete</span>
                    </button>
                  </div>
                )}
                <p className="text-[10px] text-neutral-500 text-center font-mono">
                  Allowed formats: JPG, PNG, WebP (max 10MB)
                </p>
              </div>

              {/* Date Metadata */}
              <div className="flex items-center justify-between px-1 text-[11px] text-neutral-500 font-mono pt-1">
                <span>Created: {new Date(asset.created_at).toLocaleDateString()}</span>
                {asset.updated_at && (
                  <span>Updated: {new Date(asset.updated_at).toLocaleDateString()}</span>
                )}
              </div>
            </div>

            {/* Right: Metadata & Actions */}
            <div className="flex flex-col justify-between space-y-6">
              <div className="space-y-4">
                {/* Category Pill */}
                <div>
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-cyan-500/10 border border-cyan-500/30 text-cyan-400">
                    {asset.category?.name || "Uncategorized"}
                  </span>
                </div>

                {/* Asset Title & Author */}
                <div>
                  <h1 className="text-2xl sm:text-3xl font-bold text-white tracking-tight leading-snug">
                    {asset.name}
                  </h1>
                  <p className="text-xs sm:text-sm text-neutral-400 mt-1.5 flex items-center gap-1.5">
                    <span className="text-neutral-500">by</span>
                    {asset.author ? (
                      <span className="text-neutral-200 font-medium">@{asset.author}</span>
                    ) : (
                      <span className="text-neutral-500 italic">Author unknown</span>
                    )}
                  </p>
                </div>

                {/* Description */}
                <div>
                  <h2 className="text-[11px] uppercase tracking-wider font-mono text-neutral-500 mb-1.5">
                    Description
                  </h2>
                  {asset.description ? (
                    <p className="text-xs sm:text-sm text-neutral-300 leading-relaxed whitespace-pre-wrap bg-neutral-950/40 p-3 rounded-xl border border-neutral-800">
                      {asset.description}
                    </p>
                  ) : (
                    <p className="text-xs text-neutral-600 italic">
                      No description provided for this asset.
                    </p>
                  )}
                </div>

                {/* Tags */}
                <div>
                  <h2 className="text-[11px] uppercase tracking-wider font-mono text-neutral-500 mb-2">
                    Tags
                  </h2>
                  <div className="flex flex-wrap gap-1.5">
                    {asset.tags && asset.tags.length > 0 ? (
                      asset.tags.map((t) => (
                        <span
                          key={t}
                          className="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-mono bg-neutral-800/80 text-neutral-300 border border-neutral-700/60 hover:border-neutral-600 transition-colors"
                        >
                          #{t}
                        </span>
                      ))
                    ) : (
                      <span className="text-xs text-neutral-600 italic">
                        No tags attached
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Local Path, Open Folder & External Actions */}
              <div className="space-y-4 pt-5 border-t border-neutral-800">
                {/* Local Path Card */}
                <div className="p-3.5 rounded-xl bg-neutral-950/80 border border-neutral-800 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] uppercase font-mono tracking-wider text-neutral-400 font-semibold">
                      Local Path
                    </span>

                    {/* Dynamic Status Badge */}
                    {!hasLocalPath ? (
                      <span className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-400 border border-neutral-700">
                        Not specified
                      </span>
                    ) : isCheckingStatus ? (
                      <span className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-400 border border-neutral-700">
                        <span className="h-1.5 w-1.5 rounded-full bg-neutral-400 animate-ping" />
                        Checking...
                      </span>
                    ) : fileStatus?.exists ? (
                      <span className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded-full bg-emerald-950/60 text-emerald-400 border border-emerald-800/70">
                        <svg className="w-2.5 h-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                        </svg>
                        Available
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded-full bg-rose-950/60 text-rose-400 border border-rose-800/70">
                        <svg className="w-2.5 h-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                        Missing from disk
                      </span>
                    )}
                  </div>

                  {hasLocalPath ? (
                    <div className="font-mono text-xs text-neutral-300 break-all select-all bg-neutral-900/60 p-2 rounded border border-neutral-800">
                      {asset.local_path}
                    </div>
                  ) : (
                    <p className="text-xs text-neutral-500 italic">
                      No local folder or file path configured.
                    </p>
                  )}

                  {/* Open Folder Button */}
                  {hasLocalPath && (
                    <button
                      onClick={handleOpenFolder}
                      disabled={isOpening || fileStatus?.exists === false}
                      className={`inline-flex items-center justify-center gap-2 w-full rounded-lg text-xs font-semibold py-2 px-3 transition-colors ${
                        fileStatus?.exists === false
                          ? "bg-neutral-900 text-neutral-600 border border-neutral-800 cursor-not-allowed"
                          : "bg-neutral-800 hover:bg-neutral-750 active:bg-neutral-700 text-neutral-200 hover:text-white border border-neutral-700/80 cursor-pointer shadow-sm"
                      }`}
                    >
                      {isOpening ? (
                        <>
                          <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent" />
                          <span>Opening Folder...</span>
                        </>
                      ) : (
                        <>
                          <svg className="w-4 h-4 text-cyan-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                            />
                          </svg>
                          <span>Open Folder</span>
                        </>
                      )}
                    </button>
                  )}
                </div>

                {/* Open BOOTH Action Button */}
                {asset.booth_url && (
                  <a
                    href={asset.booth_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center justify-center gap-2 w-full rounded-xl bg-gradient-to-r from-red-600 to-rose-600 hover:from-red-500 hover:to-rose-500 text-white text-xs font-semibold py-2.5 px-4 transition-all shadow-lg shadow-red-950/50 cursor-pointer"
                  >
                    <span>Open on BOOTH</span>
                    <svg
                      className="w-3.5 h-3.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                      />
                    </svg>
                  </a>
                )}
              </div>
            </div>
          </div>
        </div>
        )}

        {/* Delete Confirmation Modal */}
        <DeleteConfirmationModal
          isOpen={isDeleteModalOpen}
          assetName={asset.name}
          localPath={asset.local_path}
          isDeleting={isDeleting}
          onConfirm={handleDeleteConfirm}
          onCancel={() => setIsDeleteModalOpen(false)}
        />
      </div>
    </main>
  );
}
