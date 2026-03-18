#!/usr/bin/env bash
set -eo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

if ! command -v pnpm >/dev/null 2>&1; then
  echo "pnpm is required. Install it from https://pnpm.io/installation" >&2
  exit 1
fi

# pnpm v11 git worktree support: use the global virtual store so deps are
# symlinked rather than copied, making worktree setup near-instant.
# See: https://pnpm.io/next/git-worktrees
export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"

for dir in web docs; do
  target="$repo_root/$dir"
  if [ -f "$target/package.json" ]; then
    echo "Installing dependencies in $dir..."
    pnpm install --dir "$target"
  fi
done
