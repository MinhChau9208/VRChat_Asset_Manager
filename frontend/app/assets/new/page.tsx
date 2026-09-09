"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AssetForm } from "@/components/AssetForm";
import { Asset } from "@/lib/api";

export default function NewAssetPage() {
  const router = useRouter();

  const handleSuccess = (created: Asset) => {
    // Navigate immediately to the newly created asset detail page
    router.push(`/assets/${created.id}`);
  };

  const handleCancel = () => {
    router.push("/");
  };

  return (
    <main className="min-h-screen bg-neutral-950 text-neutral-100 p-4 sm:p-6 md:p-8">
      <div className="max-w-3xl mx-auto space-y-6">
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
        </div>

        {/* Page Title Card */}
        <div className="rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 md:p-8 backdrop-blur-md shadow-2xl space-y-6">
          <div className="border-b border-neutral-800/80 pb-5">
            <h1 className="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-3">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 text-sm">
                +
              </span>
              <span>Add New Asset</span>
            </h1>
            <p className="text-xs text-neutral-400 mt-1.5">
              Add a 3D model, avatar, hairstyle, outfit, shader, or gimmick to your
              personal VRChat asset library.
            </p>
          </div>

          {/* Asset Creation Form */}
          <AssetForm
            mode="create"
            onSubmitSuccess={handleSuccess}
            onCancel={handleCancel}
          />
        </div>
      </div>
    </main>
  );
}
