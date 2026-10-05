"use client";

import React from "react";
import Link from "next/link";
import { Box, Plus, Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ThemeToggle } from "./ThemeToggle";
import { UpdateNotice } from "./UpdateNotice";
import { cn } from "@/lib/utils";

interface HeaderProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  isConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({ searchQuery, onSearchChange, isConnected }) => {
  return (
    <header className="sticky top-0 z-30 flex h-16 w-full items-center gap-4 border-b border-border bg-background/80 px-4 backdrop-blur-md sm:px-6">
      <Link href="/" className="flex shrink-0 items-center gap-2.5">
        <div className="flex size-9 items-center justify-center rounded-lg border border-primary/30 bg-primary/10 text-primary">
          <Box className="size-5" />
        </div>
        <span className="hidden text-base font-semibold tracking-tight text-foreground sm:inline">VRChat Asset Manager</span>
      </Link>

      <div className="relative mx-auto w-full max-w-xl">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          id="search-input"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search by name, author, tag…"
          className="h-9 pl-9 pr-8"
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => onSearchChange("")}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground"
            aria-label="Clear search"
          >
            <X className="size-4" />
          </button>
        )}
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <ThemeToggle />
        <UpdateNotice />
        <Button asChild size="sm">
          <Link href="/assets/new" id="add-asset-header-btn">
            <Plus />
            <span className="hidden sm:inline">Add Asset</span>
          </Link>
        </Button>
        <span
          className={cn(
            "size-2.5 rounded-full",
            isConnected ? "bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.7)]" : "animate-pulse bg-rose-500"
          )}
          title={isConnected ? "Backend connected" : "Backend unreachable"}
        />
      </div>
    </header>
  );
};
