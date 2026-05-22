#!/usr/bin/env bash
set -euo pipefail

# rustup via the upstream curl|sh installer (not a GitHub release asset)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# a global npm tool (no structured global-npm module)
npm install -g typescript

# a one-off with no structured equivalent
mkdir -p ~/.local/state/myapp && touch ~/.local/state/myapp/initialized
