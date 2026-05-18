#!/usr/bin/env bash
# publish-scoop.sh — Updates zeshone/scoop-bucket with the new doc-agent-ai release.
# Invoked by goreleaser's publishers hook after archives + checksums are produced.
#
# Required environment variables:
#   GORELEASER_CURRENT_TAG  e.g. v3.3.0
#   GORELEASER_VERSION      e.g. 3.3.0
#   RELEASE_TOKEN           PAT with contents:write on zeshone/scoop-bucket
#
# The script is idempotent — re-running for the same tag overwrites the manifest
# with the same content and exits cleanly when there is no diff to commit.

set -euo pipefail

TAG="${GORELEASER_CURRENT_TAG:?GORELEASER_CURRENT_TAG must be set}"
VERSION="${GORELEASER_VERSION:?GORELEASER_VERSION must be set}"
TOKEN="${RELEASE_TOKEN:?RELEASE_TOKEN must be set}"

ARCHIVE="doc-agent-ai_${VERSION}_windows_amd64.zip"
URL="https://github.com/zeshone/doc-agent-ai/releases/download/${TAG}/${ARCHIVE}"

if [ ! -f dist/checksums.txt ]; then
  echo "::error::dist/checksums.txt not found — run after goreleaser build phase"
  exit 1
fi

HASH=$(awk -v a="${ARCHIVE}" '$2 == a { print $1 }' dist/checksums.txt)
if [ -z "${HASH}" ]; then
  echo "::error::SHA256 for ${ARCHIVE} not found in dist/checksums.txt"
  exit 1
fi

WORKDIR=$(mktemp -d)
trap 'rm -rf "${WORKDIR}"' EXIT

git clone --depth 1 "https://x-access-token:${TOKEN}@github.com/zeshone/scoop-bucket.git" "${WORKDIR}/scoop-bucket"
cd "${WORKDIR}/scoop-bucket"

mkdir -p bucket
cat > bucket/doc-agent-ai.json <<EOF
{
  "version": "${VERSION}",
  "description": "Multi-platform documentation workflow agent installer",
  "homepage": "https://github.com/zeshone/doc-agent-ai",
  "license": "MIT",
  "url": "${URL}",
  "hash": "${HASH}",
  "bin": "doc-agent-ai.exe",
  "checkver": {
    "github": "https://github.com/zeshone/doc-agent-ai"
  },
  "autoupdate": {
    "url": "https://github.com/zeshone/doc-agent-ai/releases/download/v\$version/doc-agent-ai_\$version_windows_amd64.zip",
    "hash": {
      "url": "https://github.com/zeshone/doc-agent-ai/releases/download/v\$version/checksums.txt"
    }
  }
}
EOF

git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add bucket/doc-agent-ai.json
git commit -m "doc-agent-ai ${VERSION}" || {
  echo "No changes to commit — bucket manifest already at ${VERSION}"
  exit 0
}
git push origin HEAD
echo "Scoop bucket updated for doc-agent-ai ${VERSION}"
