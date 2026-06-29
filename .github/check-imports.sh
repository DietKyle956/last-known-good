#!/usr/bin/env bash
# Enforce that domain packages never import infrastructure/adapter packages.
# Domain packages: agent, llm, sandbox, tools, hooks, skills, router, tui
# Infrastructure packages: store
set -euo pipefail

ROOT="github.com/DietKyle956/last-known-good"
INFRA_PKGS=("store")

domain_dirs=("agent" "llm" "sandbox" "tools" "hooks" "skills" "router" "tui")

for dir in "${domain_dirs[@]}"; do
  for infra in "${INFRA_PKGS[@]}"; do
    pattern="$ROOT/internal/$infra"
    if grep -r "$pattern" "internal/$dir/" --include="*.go" 2>/dev/null; then
      echo "FAIL: internal/$dir imports internal/$infra (domain must not import infrastructure)"
      exit 1
    fi
  done
done

echo "OK: No domain packages import infrastructure packages"
