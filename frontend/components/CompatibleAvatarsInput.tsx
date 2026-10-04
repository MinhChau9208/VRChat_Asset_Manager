"use client";

import React, { useEffect, useState } from "react";
import { Asset, CompatAvatar, getAssets } from "@/lib/api";

interface CompatibleAvatarsInputProps {
  value: CompatAvatar[];
  onChange: (value: CompatAvatar[]) => void;
  /** The asset being edited, so it is not offered as its own avatar. */
  selfId?: number;
}

const sameName = (a: string, b: string) => a.trim().toLowerCase() === b.trim().toLowerCase();

/**
 * Pick the avatars an asset works with: avatars from the library (linked by id)
 * or any other avatar by name (e.g. one you don't own yet).
 */
export const CompatibleAvatarsInput: React.FC<CompatibleAvatarsInputProps> = ({
  value,
  onChange,
  selfId,
}) => {
  const [libraryAvatars, setLibraryAvatars] = useState<Asset[]>([]);
  const [nameInput, setNameInput] = useState("");

  useEffect(() => {
    let cancelled = false;
    getAssets({ category: "Avatar", sort: "name_asc" })
      .then((avatars) => {
        if (!cancelled) setLibraryAvatars(avatars.filter((a) => a.id !== selfId));
      })
      .catch(() => {
        // The picker still works with free-text names.
      });
    return () => {
      cancelled = true;
    };
  }, [selfId]);

  const add = (entry: CompatAvatar) => {
    const duplicate = value.some(
      (v) =>
        (entry.avatar_asset_id !== null && v.avatar_asset_id === entry.avatar_asset_id) ||
        sameName(v.avatar_name, entry.avatar_name)
    );
    if (!duplicate && entry.avatar_name.trim()) {
      onChange([...value, entry]);
    }
  };

  const addByName = () => {
    const name = nameInput.trim();
    if (!name) return;
    // Typing the name of a library avatar links it instead of storing plain text.
    const match = libraryAvatars.find((a) => sameName(a.name, name));
    add(match ? { avatar_asset_id: match.id, avatar_name: match.name } : { avatar_asset_id: null, avatar_name: name });
    setNameInput("");
  };

  const unpicked = libraryAvatars.filter(
    (a) => !value.some((v) => v.avatar_asset_id === a.id || sameName(v.avatar_name, a.name))
  );

  return (
    <div className="space-y-2">
      <label className="block text-xs font-medium text-neutral-300">
        Compatible Avatars ({value.length})
      </label>

      {value.length === 0 ? (
        <p className="text-xs text-neutral-500 italic">
          Not set. Add the avatars this asset is made for.
        </p>
      ) : (
        <div className="flex flex-wrap gap-1.5">
          {value.map((v, idx) => (
            <span
              key={`${v.avatar_asset_id ?? "name"}-${v.avatar_name}`}
              className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium border ${
                v.avatar_asset_id !== null
                  ? "bg-violet-950/50 text-violet-200 border-violet-800/70"
                  : "bg-neutral-800 text-neutral-200 border-neutral-700/80"
              }`}
              title={v.avatar_asset_id !== null ? "Avatar in your library" : "Avatar not in your library"}
            >
              <span>👤</span>
              <span>{v.avatar_name}</span>
              <button
                type="button"
                onClick={() => onChange(value.filter((_, i) => i !== idx))}
                className="text-neutral-400 hover:text-rose-400 transition-colors cursor-pointer p-0.5"
                title={`Remove ${v.avatar_name}`}
              >
                ✕
              </button>
            </span>
          ))}
        </div>
      )}

      <div className="flex flex-col sm:flex-row gap-2">
        {unpicked.length > 0 && (
          <select
            value=""
            onChange={(e) => {
              const avatar = libraryAvatars.find((a) => a.id === Number(e.target.value));
              if (avatar) add({ avatar_asset_id: avatar.id, avatar_name: avatar.name });
            }}
            className="sm:w-56 rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-1.5 text-xs sm:text-sm text-white focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 cursor-pointer"
          >
            <option value="">+ From your library…</option>
            {unpicked.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        )}
        <div className="flex flex-1 gap-2">
          <input
            type="text"
            value={nameInput}
            onChange={(e) => setNameInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                addByName();
              }
            }}
            placeholder="Other avatar name (e.g. Manuka) and press Enter"
            className="flex-1 rounded-lg border border-neutral-800 bg-neutral-950/80 px-3 py-1.5 text-xs sm:text-sm text-white placeholder-neutral-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
          />
          <button
            type="button"
            onClick={addByName}
            className="px-3 py-1.5 rounded-lg bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium border border-neutral-700 transition-colors cursor-pointer shrink-0"
          >
            + Add
          </button>
        </div>
      </div>
    </div>
  );
};
