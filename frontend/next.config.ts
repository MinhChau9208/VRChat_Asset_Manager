import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Build to plain HTML/JS (frontend/out) so the Go binary can embed and serve
  // the UI; see scripts/build-release.ps1.
  output: "export",
  // Hide the floating "N" dev-tools button (it only exists under `next dev`;
  // build errors are still shown as an overlay).
  devIndicators: false,
};

export default nextConfig;
