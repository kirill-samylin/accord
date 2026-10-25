#!/bin/sh
# Build the Linux x64 Next bundle from the committed tree; pairs with build-release.sh.
# Output: .local/releases/<revision>-next (package.py manifest, Linux Node, standalone app).
set -eu
umask 022
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo"
if ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git ls-files --others --exclude-standard)" ]; then
  echo 'Commit the reviewed tree before building a release.' >&2
  exit 1
fi
revision=$(git rev-parse HEAD)
node_bin=${ACCORD_NODE:-node}
if [ "$("$node_bin" --version)" != v24.19.0 ]; then
  echo 'Next production build requires Node v24.19.0 (set ACCORD_NODE).' >&2
  exit 1
fi
# Linux runtime shipped inside the bundle. The official archive hash is pinned
# here; package.py re-checks the extracted binary against NODE_SHA256.
NODE_VERSION=v24.19.0
ARCHIVE_SHA256=14b342e71204f811bde6153be8e04b62aef63c236fef92b55f9c83154b409647
NODE_SHA256=bc17c508ffeed0ec622934f9b7fa72f8e78da65350e63c3eceb56fa688aa5e12
linux_node=${ACCORD_LINUX_NODE:-$repo/.local/toolchains/node-$NODE_VERSION-linux-x64/bin/node}
if [ ! -x "$linux_node" ]; then
  archive="$repo/.local/toolchains/node-$NODE_VERSION-linux-x64.tar.xz"
  mkdir -p "$repo/.local/toolchains"
  curl --fail --silent --show-error --location \
    "https://nodejs.org/dist/$NODE_VERSION/node-$NODE_VERSION-linux-x64.tar.xz" -o "$archive"
  printf '%s  %s\n' "$ARCHIVE_SHA256" "$archive" | shasum -a 256 -c - >/dev/null
  tar -xJf "$archive" -C "$repo/.local/toolchains"
fi
output="$repo/.local/releases/$revision-next"
if [ -e "$output" ]; then
  echo "Next bundle already exists: $output" >&2
  exit 1
fi
export NEXT_TELEMETRY_DISABLED=1
# Production upstream is the Go service on 7892; package.py rejects the dev default.
(cd web && ACCORD_GO_ORIGIN=http://127.0.0.1:7892 "$node_bin" node_modules/next/dist/bin/next build)
python3 infra/accord/next/package.py --revision "$revision" --node "$linux_node" \
  --node-sha256 "$NODE_SHA256" --output "$output"
printf 'Linux x64 Next bundle prepared from %s at %s\n' "$revision" "$output"
