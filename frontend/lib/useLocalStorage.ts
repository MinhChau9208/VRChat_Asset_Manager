"use client";

import { useCallback, useSyncExternalStore } from "react";

const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

function read(key: string): string | null {
  try {
    return window.localStorage.getItem(key);
  } catch {
    return null; // private mode or blocked storage
  }
}

/**
 * A per-browser UI preference (card size, view mode). The server render and
 * first client render use the fallback, so there is no hydration mismatch.
 */
export function useLocalStorage<T extends string>(
  key: string,
  fallback: T,
  allowed: readonly T[]
): [T, (value: T) => void] {
  const raw = useSyncExternalStore(
    subscribe,
    () => read(key),
    () => null
  );
  const value = raw !== null && (allowed as readonly string[]).includes(raw) ? (raw as T) : fallback;

  const setValue = useCallback(
    (next: T) => {
      try {
        window.localStorage.setItem(key, next);
      } catch {
        // Preference is simply not remembered.
      }
      listeners.forEach((l) => l());
    },
    [key]
  );

  return [value, setValue];
}
