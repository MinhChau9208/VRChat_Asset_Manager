import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "VRChat Asset Manager",
  description: "Local-first personal asset catalog and management tool for VRChat",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-neutral-950 text-neutral-100 antialiased selection:bg-cyan-500/20 selection:text-cyan-200">
        {children}
      </body>
    </html>
  );
}
