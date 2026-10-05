"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { FolderOpen, Loader2, Plus, ScanSearch, Sparkles } from "lucide-react";
import { getScannerConfig, pickFolder, runScan, saveScannerConfig } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useI18n } from "@/lib/i18n";

/**
 * Shown instead of the empty grid while the library has no assets: leads a
 * first-time user from "pick your asset folder" to the Review screen.
 */
export function WelcomePanel({ draftCount }: { draftCount: number }) {
  const router = useRouter();
  const [folder, setFolder] = useState("");
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { t } = useI18n();

  // A scan already ran; the drafts only need confirming.
  if (draftCount > 0) {
    return (
      <Panel>
        <h2 className="text-xl font-semibold">
          {t.welcome.waiting(draftCount)}
        </h2>
        <p className="max-w-lg text-sm text-muted-foreground">
          {t.welcome.waitingText}
        </p>
        <Button asChild>
          <Link href="/review">
            <ScanSearch /> {t.welcome.openReview}
          </Link>
        </Button>
      </Panel>
    );
  }

  const browse = async () => {
    const selected = await pickFolder();
    if (selected) setFolder(selected);
  };

  const scan = async () => {
    const root = folder.trim();
    if (!root) return;
    setScanning(true);
    setError(null);
    try {
      const config = await getScannerConfig();
      if (!config.roots.some((r) => r.toLowerCase() === root.toLowerCase())) {
        await saveScannerConfig({ ...config, roots: [...config.roots, root] });
      }
      await runScan();
      router.push("/review");
    } catch (err) {
      setError(err instanceof Error ? err.message : t.welcome.scanFailed);
      setScanning(false);
    }
  };

  return (
    <Panel>
      <h2 className="text-xl font-semibold">{t.welcome.title}</h2>
      <p className="max-w-lg text-sm text-muted-foreground">{t.welcome.intro}</p>

      <form
        className="flex w-full max-w-xl flex-col gap-2 sm:flex-row"
        onSubmit={(e) => {
          e.preventDefault();
          scan();
        }}
      >
        <Input
          value={folder}
          onChange={(e) => setFolder(e.target.value)}
          placeholder="D:\VRChat Assets"
          aria-label={t.welcome.folder}
          className="font-mono text-sm"
          disabled={scanning}
        />
        <Button type="button" variant="outline" onClick={browse} disabled={scanning}>
          <FolderOpen /> {t.welcome.browse}
        </Button>
        <Button type="submit" disabled={scanning || !folder.trim()}>
          {scanning ? <Loader2 className="animate-spin" /> : <ScanSearch />}
          {scanning ? t.welcome.scanning : t.welcome.scan}
        </Button>
      </form>
      {error && <p className="max-w-xl text-sm text-destructive">{error}</p>}

      <p className="max-w-lg text-xs text-muted-foreground">{t.welcome.safe}</p>

      <div className="text-sm text-muted-foreground">
        {t.welcome.or}{" "}
        <Link href="/assets/new" className="inline-flex items-center gap-1 text-primary hover:underline">
          <Plus className="size-3.5" /> {t.welcome.addByHand}
        </Link>
      </div>
    </Panel>
  );
}

function Panel({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-[360px] flex-col items-center justify-center gap-4 rounded-2xl border border-dashed border-border bg-card/20 p-8 text-center">
      <div className="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
        <Sparkles className="size-6" />
      </div>
      {children}
    </div>
  );
}
