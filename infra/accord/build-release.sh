#!/bin/sh
# Build only the committed, reviewed tree; never embed credentials or build paths.
set -eu
umask 077
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$repo"
if ! git diff --quiet || ! git diff --cached --quiet || [ -n "$(git ls-files --others --exclude-standard)" ]; then
  echo 'Commit the reviewed tree before building a release.' >&2
  exit 1
fi
revision=$(git rev-parse HEAD)
go_bin=${ACCORD_GO:-"$repo/.local/toolchains/go/bin/go"}
if [ ! -x "$go_bin" ]; then
  echo 'Set ACCORD_GO to the pinned Go1.27.2 executable.' >&2
  exit 1
fi
if [ "$("$go_bin" version | cut -d ' ' -f 3)" != go1.27.2 ]; then
  echo 'Release build requires Go1.27.2.' >&2
  exit 1
fi
release="$repo/.local/releases/$revision"
mkdir -p "$repo/.local/releases"
mkdir "$release"
mkdir "$release/infra"
export GOTOOLCHAIN=local
export GOCACHE="$repo/.local/toolchains/cache/build"
export GOMODCACHE="$repo/.local/toolchains/cache/mod"
export GOTMPDIR="$repo/.local/toolchains/tmp"
mkdir -p "$GOTMPDIR"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 "$go_bin" build -trimpath -buildvcs=true -o "$release/agent-board" ./cmd/agent-board
cp infra/accord/*.py infra/accord/*.service infra/accord/*.timer "$release/infra/"
printf '%s\n' "$revision" > "$release/SOURCE"
python3 - "$release" <<'PY'
import hashlib, pathlib, sys
root = pathlib.Path(sys.argv[1])
with (root / 'SHA256SUMS').open('x', encoding='ascii') as output:
    for path in sorted(root.rglob('*')):
        if path.is_file() and path.name != 'SHA256SUMS':
            value = hashlib.sha256(path.read_bytes()).hexdigest()
            output.write(value + '  ' + str(path.relative_to(root)) + '\n')
PY
printf 'Linux/amd64 release prepared from %s at %s\n' "$revision" "$release"
