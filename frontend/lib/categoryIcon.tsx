import { createElement } from "react";
import {
  Box,
  Brush,
  Ear,
  Eye,
  Folder,
  Footprints,
  Gem,
  Globe,
  Layers,
  type LucideIcon,
  type LucideProps,
  Music,
  Package,
  PersonStanding,
  Scissors,
  Shirt,
  Smile,
  Sparkles,
  UserRound,
  Wrench,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  avatar: UserRound,
  outfit: Shirt,
  clothes: Shirt,
  shoes: Footprints,
  hair: Scissors,
  accessory: Gem,
  "ears & tail": Ear,
  face: Smile,
  eyes: Eye,
  expression: Smile,
  makeup: Brush,
  gimmick: Sparkles,
  prop: Package,
  animation: PersonStanding,
  "texture & material": Layers,
  "tool & shader": Wrench,
  world: Globe,
  audio: Music,
  other: Box,
};

/** Icon for a category name; custom categories get a folder. */
export function categoryIcon(name: string | undefined): LucideIcon {
  return (name && ICONS[name.toLowerCase()]) || Folder;
}

/** Renders the icon of a category; a stable component, safe to use in JSX. */
export function CategoryIcon({ name, ...props }: { name: string | undefined } & LucideProps) {
  return createElement(categoryIcon(name), props);
}
