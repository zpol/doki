#!/usr/bin/env bash
# Set professional About metadata for github.com/zpol/doki
# Requires: gh auth login
set -euo pipefail

REPO="${1:-zpol/doki}"

gh repo edit "$REPO" \
  --description "Terminal-native Docker Operations Cockpit — keyboard-driven TUI for real Docker Engine" \
  --homepage "https://github.com/zpol/doki" \
  --add-topic go \
  --add-topic golang \
  --add-topic docker \
  --add-topic tui \
  --add-topic terminal \
  --add-topic cli \
  --add-topic devops \
  --add-topic containers \
  --add-topic bubbletea \
  --add-topic monitoring \
  --add-topic security \
  --add-topic apache2

echo "Updated About metadata for $REPO"
gh repo view "$REPO" --json description,repositoryTopics,url
