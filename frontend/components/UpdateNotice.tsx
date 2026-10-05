"use client";

import React, { useEffect, useState } from "react";
import { ArrowUpCircle } from "lucide-react";
import { VersionInfo, getVersionInfo } from "@/lib/api";
import { Button } from "@/components/ui/button";

/** A small link to the release page when a newer version is out. */
export function UpdateNotice() {
  const [info, setInfo] = useState<VersionInfo | null>(null);

  useEffect(() => {
    getVersionInfo()
      .then(setInfo)
      .catch(() => {}); // offline or an older backend: show nothing
  }, []);

  if (!info?.update_available || !info.url) return null;

  return (
    <Button asChild variant="outline" size="sm" className="border-emerald-500/40 text-emerald-700 dark:text-emerald-300">
      <a
        href={info.url}
        target="_blank"
        rel="noopener noreferrer"
        title={`You have ${info.version}. Download ${info.latest}, then replace the exe and keep the data folder.`}
      >
        <ArrowUpCircle />
        <span className="hidden sm:inline">Update {info.latest}</span>
      </a>
    </Button>
  );
}
