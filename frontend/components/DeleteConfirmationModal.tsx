"use client";

import React from "react";

interface DeleteConfirmationModalProps {
  isOpen: boolean;
  assetName: string;
  localPath?: string;
  isDeleting: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export const DeleteConfirmationModal: React.FC<DeleteConfirmationModalProps> = ({
  isOpen,
  assetName,
  localPath,
  isDeleting,
  onConfirm,
  onCancel,
}) => {
  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-dialog-title"
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-150"
    >
      <div className="w-full max-w-md rounded-2xl border border-neutral-800 bg-neutral-900 p-6 shadow-2xl space-y-5">
        {/* Header with warning icon */}
        <div className="flex items-start gap-3.5">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-400 text-lg">
            ⚠️
          </div>
          <div>
            <h2
              id="delete-dialog-title"
              className="text-base font-bold text-white tracking-tight"
            >
              Delete Asset
            </h2>
            <p className="text-xs text-neutral-400 mt-0.5">
              Are you sure you want to remove{" "}
              <span className="text-white font-semibold">&ldquo;{assetName}&rdquo;</span>{" "}
              from your library?
            </p>
          </div>
        </div>

        {/* Safety Guarantee Notice */}
        <div className="rounded-xl border border-emerald-900/40 bg-emerald-950/20 p-3.5 text-xs space-y-1.5">
          <div className="flex items-center gap-2 text-emerald-400 font-medium">
            <span>🛡️</span>
            <span>Your original VRChat files will NOT be deleted</span>
          </div>
          <p className="text-neutral-400 text-[11px] leading-relaxed">
            Deleting an asset removes its catalog database record and application-owned
            preview image only. The local files and folders on your computer remain
            completely untouched.
          </p>
          {localPath && (
            <div className="mt-1 font-mono text-[10px] text-neutral-400 break-all bg-black/40 p-1.5 rounded border border-neutral-800">
              {localPath}
            </div>
          )}
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-3 pt-1">
          <button
            type="button"
            onClick={onCancel}
            disabled={isDeleting}
            className="px-4 py-2 rounded-xl text-xs font-semibold text-neutral-300 hover:text-white bg-neutral-800 hover:bg-neutral-750 border border-neutral-700 transition-colors cursor-pointer disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isDeleting}
            className="inline-flex items-center justify-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold text-white bg-rose-600 hover:bg-rose-500 active:bg-rose-700 shadow-lg shadow-rose-950/50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isDeleting ? (
              <>
                <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                <span>Deleting...</span>
              </>
            ) : (
              <span>Delete Asset</span>
            )}
          </button>
        </div>
      </div>
    </div>
  );
};
