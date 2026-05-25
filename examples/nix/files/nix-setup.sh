#!/bin/sh
# Optional nix setup script, run by the `nix` module after install with the nix
# profile already sourced (so `nix`, NIX_PATH and the SSL cert env are set).
#
# This is the escape hatch for anything the declarative `packages:` list does not
# cover: applying a flake, enabling a channel, installing home-manager, etc. It
# runs whenever the nix phase runs, so keep every step idempotent.
set -eu

# Example: install a package from a specific flake only if it is not present yet.
# `nix profile install` is itself close to idempotent, but guarding keeps re-runs
# quiet and fast.
if ! command -v jq >/dev/null 2>&1; then
  nix profile install nixpkgs#jq
fi

# Example: pull in a flake-defined dev shell or apply home-manager here, e.g.
#   nix run home-manager/master -- switch --flake ~/dotfiles
#
# Anything you would normally type at a nix prompt works here.

echo "nix setup script complete"
