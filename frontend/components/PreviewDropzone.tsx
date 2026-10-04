"use client";

import React, { useEffect, useRef, useState } from "react";
import { ImagePlus, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

const ACCEPTED = ["image/jpeg", "image/png", "image/webp"];
const MAX_SIZE = 10 * 1024 * 1024;

/** Returns an error message, or null if the file can be used as a preview. */
export function validatePreviewFile(file: File): string | null {
  if (!ACCEPTED.includes(file.type)) return "Only JPEG, PNG and WebP images can be used.";
  if (file.size > MAX_SIZE) return "The image is larger than 10MB.";
  return null;
}

type Handlers = { onFile: (file: File) => void; onError?: (message: string) => void; busy: boolean };

function acceptWith(h: Handlers, file: File | undefined | null) {
  if (!file || h.busy) return;
  const problem = validatePreviewFile(file);
  if (problem) h.onError?.(problem);
  else h.onFile(file);
}

function isEditable(target: EventTarget | null) {
  const el = target as HTMLElement | null;
  return Boolean(el && (el.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName)));
}

interface PreviewDropzoneProps {
  /** Current image URL, if any. */
  src?: string | null;
  onFile: (file: File) => void;
  onError?: (message: string) => void;
  busy?: boolean;
  /** Listen for Ctrl+V anywhere on the page (except in text fields). */
  pasteAnywhere?: boolean;
  className?: string;
  children?: React.ReactNode;
}

/**
 * Square preview area: click to pick a file, drop an image on it, or paste one
 * from the clipboard (e.g. a screenshot of a BOOTH page).
 */
export const PreviewDropzone: React.FC<PreviewDropzoneProps> = ({
  src,
  onFile,
  onError,
  busy = false,
  pasteAnywhere = false,
  className,
  children,
}) => {
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const [imgError, setImgError] = useState<string | null>(null);

  // Keep the latest callbacks for the global paste listener.
  const handlers = useRef<Handlers>({ onFile, onError, busy });
  useEffect(() => {
    handlers.current = { onFile, onError, busy };
  });

  const accept = (file: File | undefined | null) => acceptWith(handlers.current, file);

  useEffect(() => {
    if (!pasteAnywhere) return;
    const onPaste = (e: ClipboardEvent) => {
      if (isEditable(e.target)) return;
      const file = Array.from(e.clipboardData?.files ?? []).find((f) => f.type.startsWith("image/"));
      if (file) {
        e.preventDefault();
        acceptWith(handlers.current, file);
      }
    };
    document.addEventListener("paste", onPaste);
    return () => document.removeEventListener("paste", onPaste);
  }, [pasteAnywhere]);

  const showImage = Boolean(src) && imgError !== src;

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => !busy && inputRef.current?.click()}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          inputRef.current?.click();
        }
      }}
      onPaste={(e) => accept(Array.from(e.clipboardData.files).find((f) => f.type.startsWith("image/")))}
      onDragOver={(e) => {
        e.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDragging(false);
        accept(e.dataTransfer.files[0]);
      }}
      className={cn(
        "group relative flex aspect-square w-full cursor-pointer items-center justify-center overflow-hidden rounded-xl border bg-muted/30 outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring",
        dragging ? "border-primary bg-primary/10" : "border-border hover:border-primary/40",
        className
      )}
      title="Click, drop or paste an image"
    >
      {showImage && src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={src} alt="" className="size-full object-cover" onError={() => setImgError(src)} />
      ) : (
        <div className="flex flex-col items-center gap-2 px-4 text-center text-muted-foreground">
          <ImagePlus className="size-8" strokeWidth={1.5} />
          <span className="text-sm">Click, drop or paste an image</span>
          <span className="text-xs text-muted-foreground/70">JPEG, PNG or WebP · max 10MB</span>
        </div>
      )}

      {showImage && !busy && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/55 opacity-0 transition-opacity group-hover:opacity-100">
          <span className="flex items-center gap-2 rounded-md bg-black/60 px-3 py-1.5 text-sm text-white">
            <ImagePlus className="size-4" /> Replace (click, drop or paste)
          </span>
        </div>
      )}
      {busy && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60">
          <Loader2 className="size-6 animate-spin text-white" />
        </div>
      )}
      {children}

      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED.join(",")}
        className="hidden"
        onChange={(e) => {
          accept(e.target.files?.[0]);
          e.target.value = "";
        }}
      />
    </div>
  );
};
