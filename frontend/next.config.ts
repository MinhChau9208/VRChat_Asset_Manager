import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Hide the floating "N" dev-tools button (it only exists under `next dev`;
  // build errors are still shown as an overlay).
  devIndicators: false,
};

export default nextConfig;
