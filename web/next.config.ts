import type { NextConfig } from "next";
import { validateGoOrigin } from "./src/lib/upstream";
const origin = validateGoOrigin(
  process.env.ACCORD_GO_ORIGIN ?? "http://127.0.0.1:3101",
);
const config: NextConfig = {
  distDir: ".local/next",
  poweredByHeader: false,
  // Go owns /web/ canonical paths; Next normalization otherwise redirects JSON
  // reads before the fixed rewrite and conflicts with redirect:error clients.
  skipTrailingSlashRedirect: true,
  // Next also clones external rewrite bodies. Keep the full Go 50 MiB file plus
  // multipart envelope and over-limit boundary byte; Go remains authoritative.
  // Direct /web ingress to Go avoids this bounded Node buffer in production.
  experimental: {
    proxyTimeout: 130000,
    proxyClientMaxBodySize: 51 * 1024 * 1024,
  },
  async rewrites() {
    return [{ source: "/web/:path*", destination: `${origin}/web/:path*` }];
  },
};
export default config;
