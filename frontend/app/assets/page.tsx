"use client";

import { Suspense } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { AssetDetail } from "@/components/AssetDetail";
import { useI18n } from "@/lib/i18n";

function AssetDetailView() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const id = searchParams.get("id") ?? "";
  const { t } = useI18n();

  return (
    <main className="min-h-screen p-4 sm:p-6 md:p-8">
      <div className="mx-auto max-w-5xl space-y-6">
        <Link href="/" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary">
          <ArrowLeft className="size-4" /> {t.common.library}
        </Link>
        <div className="rounded-2xl border border-border bg-card/40 p-5 sm:p-8">
          {/* Keyed by id so navigating between assets starts fresh. */}
          <AssetDetail key={id} assetId={id} variant="page" onDeleted={() => router.push("/")} />
        </div>
      </div>
    </main>
  );
}

export default function AssetDetailPage() {
  return (
    <Suspense>
      <AssetDetailView />
    </Suspense>
  );
}
