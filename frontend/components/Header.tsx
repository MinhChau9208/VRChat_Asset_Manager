"use client";

import React from "react";
import Link from "next/link";

interface HeaderProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  isConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  searchQuery,
  onSearchChange,
  isConnected,
}) => {
  return (
    <header className="sticky top-0 z-30 flex h-16 w-full items-center justify-between border-b border-neutral-800 bg-neutral-950/80 px-4 sm:px-6 backdrop-blur-md">
      {/* Brand */}
      <div className="flex items-center gap-3">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 font-bold text-sm tracking-wider">
          VR
        </div>
        <div>
          <h1 className="text-base sm:text-lg font-bold tracking-tight text-white flex items-center gap-2">
            VRChat Asset Manager
          </h1>
        </div>
      </div>

      {/* Search Input */}
      <div className="flex-1 max-w-md mx-4">
        <div className="relative">
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-neutral-400">
            <svg
              className="h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
          </div>
          <input
            id="search-input"
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search assets by name, author, or description..."
            className="w-full rounded-lg border border-neutral-800 bg-neutral-900/90 py-2 pl-9 pr-8 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 transition-all"
          />
          {searchQuery && (
            <button
              onClick={() => onSearchChange("")}
              className="absolute inset-y-0 right-0 flex items-center pr-2.5 text-neutral-400 hover:text-white"
              title="Clear search"
            >
              <svg
                className="h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* Right Actions: Add Asset Button & Connection Indicator */}
      <div className="flex items-center gap-3">
        <Link
          href="/assets/new"
          id="add-asset-header-btn"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 active:bg-cyan-600 text-white text-xs font-semibold shadow-sm shadow-cyan-950/40 transition-colors cursor-pointer shrink-0"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
          </svg>
          <span className="hidden sm:inline">Add Asset</span>
        </Link>

        <div
          className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${
            isConnected
              ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"
              : "bg-rose-500/10 text-rose-400 border-rose-500/30"
          }`}
          title={isConnected ? "Backend connected" : "Backend unreachable"}
        >
          <span
            className={`h-1.5 w-1.5 rounded-full ${
              isConnected
                ? "bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]"
                : "bg-rose-400 animate-pulse"
            }`}
          />
          <span className="hidden sm:inline">
            {isConnected ? "Connected" : "Offline"}
          </span>
        </div>
      </div>
    </header>
  );
};
