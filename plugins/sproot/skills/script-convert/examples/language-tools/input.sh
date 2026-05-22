#!/usr/bin/env bash
set -euo pipefail

# python CLIs
uv tool install ruff
uv tool install pre-commit

# go tools
go install golang.org/x/tools/cmd/goimports@latest

# rust crates and components
cargo install ripgrep
cargo install --locked --version 0.9.72 cargo-nextest
rustup component add clippy rustfmt

# node package manager
corepack enable
corepack prepare pnpm@latest --activate
