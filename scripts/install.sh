#!/usr/bin/env bash
set -euo pipefail

repo="wewesss/ghstats"

resolve_tag() {
  local version="${GHSTATS_VERSION:-}" ref="${ACTION_REF:-}" major tags
  if [ -n "$version" ] && [ "$version" != "latest" ]; then
    echo "$version"
    return
  fi
  if [ "$version" = "latest" ]; then
    gh release view --repo "$repo" --json tagName --jq .tagName
    return
  fi
  if [[ "$ref" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "$ref"
    return
  fi
  if [[ "$ref" =~ ^v[0-9]+$ ]]; then
    major="$ref"
    tags="$(gh release list --repo "$repo" --exclude-drafts --exclude-pre-releases --limit 100 --json tagName \
      --jq ".[] | select(.tagName | startswith(\"${major}.\")) | .tagName")"
    if [ -z "$tags" ]; then
      echo "ghstats: no published release found for ${major}.x.y in ${repo}" >&2
      return 1
    fi
    printf '%s\n' "$tags" | sort -V | tail -n 1
    return
  fi
  echo "::warning::ghstats: action ref '${ref}' is not a release tag, installing the latest release" >&2
  gh release view --repo "$repo" --json tagName --jq .tagName
}

if [ -n "${GHSTATS_RESOLVE_ONLY:-}" ]; then
  resolve_tag
  exit 0
fi

case "$RUNNER_OS_NAME" in
  Linux) goos=linux ;;
  macOS) goos=darwin ;;
  *) echo "ghstats: unsupported runner OS '$RUNNER_OS_NAME' (use ubuntu-latest or macos-latest)" >&2; exit 1 ;;
esac
case "$RUNNER_ARCH_NAME" in
  X64) goarch=amd64 ;;
  ARM64) goarch=arm64 ;;
  *) echo "ghstats: unsupported runner architecture '$RUNNER_ARCH_NAME'" >&2; exit 1 ;;
esac

tag="$(resolve_tag)"
archive="ghstats_${tag#v}_${goos}_${goarch}.tar.gz"
dl="$RUNNER_TEMP/ghstats-download"
rm -rf "$dl"
mkdir -p "$dl"
gh release download "$tag" --repo "$repo" --dir "$dl" --pattern "$archive" --pattern checksums.txt
cd "$dl"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum -c --ignore-missing checksums.txt
else
  shasum -a 256 -c --ignore-missing checksums.txt
fi
bin="$RUNNER_TEMP/ghstats"
rm -rf "$bin"
mkdir -p "$bin"
tar -xzf "$archive" -C "$bin"
echo "$bin" >> "$GITHUB_PATH"
