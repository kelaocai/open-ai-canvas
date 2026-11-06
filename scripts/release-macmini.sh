#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
target="${RIGHTMOVIE_SSH_TARGET:-zech@100.66.1.12}"
remote=/Volumes/workspace/docker-data/rightmovie/deployment
if [[ -n "$(git status --porcelain)" ]]; then
    echo 'Commit the working tree before releasing' >&2
    exit 1
fi
bun_bin="${BUN_BIN:-bun}"
export PATH="$(dirname "$(command -v "$bun_bin")"):$PATH"
deployed=$(ssh "$target" "cat '$remote/deployed-sha' 2>/dev/null || true")
checks=(--all)
if [[ "$deployed" =~ ^[0-9a-f]{40}$ ]] && git cat-file -e "$deployed^{commit}" 2>/dev/null; then
    checks=(--base "$deployed")
fi
"$bun_bin" scripts/verify-before-push.mjs --jobs 3 "${checks[@]}"
sha=$(git rev-parse HEAD)
backend="rightmovie-backend:sha-$sha"
web="rightmovie-web:sha-$sha"
agent="rightmovie-yingce-agent:sha-$sha"
docker build --platform linux/arm64 -f backend/Dockerfile --build-arg GOPROXY=https://goproxy.cn,direct --build-arg BUILD_COMMIT="$sha" -t "$backend" .
docker build --platform linux/arm64 -f Dockerfile --build-arg BUILD_COMMIT="$sha" --build-arg CANVAS_BASE_PATH=/rightmovie/ --build-arg VITE_CANVAS_BACKEND_URL=/rightmovie/api -t "$web" .
docker build --platform linux/arm64 -f yingce-agent/Dockerfile -t "$agent" .
mkdir -p .local/cache
stage=$(mktemp -d "$PWD/.local/cache/macmini-transfer.XXXXXX")
trap 'rm -rf "$stage"' EXIT
docker save "$backend" "$web" "$agent" | tar -xf - -C "$stage"
ssh "$target" "mkdir -p '$remote/.image-transfer'"
# Docker archives use content-addressed layer names. rsync reuses unchanged files.
# Docker archive metadata can retain identical sizes and timestamps across tags.
# Compare contents so a new manifest is never skipped by rsync's quick check.
rsync -az --checksum --delete --stats "$stage/" "$target:$remote/.image-transfer/"
scp docker-compose.external.yml "$target:$remote/compose.yaml"
scp scripts/deploy-local.py "$target:$remote/deploy-local.py"
ssh "$target" "set -eu; /usr/bin/tar -cf - -C '$remote/.image-transfer' . | /usr/local/bin/docker load; /usr/bin/python3 '$remote/deploy-local.py' '$sha'"
