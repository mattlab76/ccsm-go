#!/usr/bin/env bash
# Syncs docs/WIKI.md to the mattlab.at (and optionally GitHub) wiki
# repos. Forgejo and GitHub wikis are separate git repos under
# <repo>.wiki.git with their own default branch.
#
# Usage: ./scripts/wiki-sync.sh
#        make wiki-sync

set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -f docs/WIKI.md ]; then
    echo "docs/WIKI.md not found" >&2
    exit 1
fi

# Wiki targets: (remote-url, label). Add the GitHub wiki here once
# the Wiki feature is enabled in repo settings.
TARGETS=(
    "ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.wiki.git mattlab.at"
    # "git@github.com:mattlab76/ccsm-go.wiki.git github.com"
)

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "untracked")

for entry in "${TARGETS[@]}"; do
    url="${entry%% *}"
    label="${entry#* }"
    echo "==> Syncing to $label ($url)"

    dest="$TMP/$label"
    if ! git clone --quiet "$url" "$dest" 2>/dev/null; then
        echo "    clone failed; wiki feature probably disabled in repo settings" >&2
        continue
    fi

    cp docs/WIKI.md "$dest/Home.md"

    pushd "$dest" >/dev/null
    if git diff --quiet; then
        echo "    no changes, skipping"
        popd >/dev/null
        continue
    fi

    git add Home.md
    git -c user.email="$(git -C "$OLDPWD" config user.email)" \
        -c user.name="$(git -C "$OLDPWD" config user.name)" \
        commit --quiet -m "wiki: mirror docs/WIKI.md from ccsm-go@$GIT_SHA"

    # Forgejo wikis default to master; GitHub modern wikis default to
    # master too. Push to whichever the remote already has, fall back
    # to both so the wiki UI's HEAD ref doesn't dangle.
    if git ls-remote --heads origin master | grep -q master; then
        git push --quiet origin HEAD:master
    fi
    if git ls-remote --heads origin main | grep -q main; then
        git push --quiet origin HEAD:main
    fi
    # First-time push: create both branches so HEAD resolves either way.
    if ! git ls-remote --heads origin master | grep -q master && \
       ! git ls-remote --heads origin main   | grep -q main; then
        git push --quiet origin HEAD:master
        git push --quiet origin HEAD:main
    fi

    echo "    pushed"
    popd >/dev/null
done

echo
echo "Wiki sync complete."
