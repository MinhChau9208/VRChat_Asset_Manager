"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { AssetForm } from "@/components/AssetForm";
import { Asset } from "@/lib/api";

export default function NewAssetPage() {
  const router = useRouter();

  return (
    <main className="min-h-screen p-4 sm:p-6 md:p-8">
      <div className="mx-auto max-w-3xl space-y-6">
        <Link href="/" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary">
          <ArrowLeft className="size-4" /> Library
        </Link>
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Add asset</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Paste a BOOTH link and press <span className="text-foreground">Fetch from BOOTH</span> to fill most fields.
          </p>
        </div>
        {/* Open the new asset in the library drawer. */}
        <AssetForm
          mode="create"
          onSubmitSuccess={(created: Asset) => router.push(`/?asset=${created.id}`)}
          onCancel={() => router.push("/")}
        />
      </div>
    </main>
  );
}
