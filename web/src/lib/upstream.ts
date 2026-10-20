// Only a deployment-owned literal loopback origin may receive browser cookies.
export function validateGoOrigin(value: string): string {
  const url = new URL(value);
  if (
    url.protocol !== "http:" ||
    !["127.0.0.1", "[::1]"].includes(url.hostname) ||
    !url.port ||
    url.username ||
    url.password ||
    url.pathname !== "/" ||
    url.search ||
    url.hash
  ) {
    throw new Error(
      "ACCORD_GO_ORIGIN must be an http literal-loopback origin with an explicit port",
    );
  }
  return url.origin;
}

export function browserRewrites(origin: string) {
  const upstream = validateGoOrigin(origin);
  // Next removes the slash before an empty wildcard. A literal first rule
  // preserves Go's /web/ root and its existing view/query validation.
  return [
    { source: "/web/", destination: `${upstream}/web/` },
    { source: "/web/:path*", destination: `${upstream}/web/:path*` },
  ];
}
