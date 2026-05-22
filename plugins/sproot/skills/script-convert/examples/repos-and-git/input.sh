#!/usr/bin/env bash
set -euo pipefail

# git identity and a preference
git config --global user.name "Ada Lovelace"
git config --global user.email "ada@example.com"
git config --global init.defaultBranch main
git config --global pull.rebase true

# clone repos
mkdir -p ~/repos
git clone git@github.com:justanotherspy/garlic.git ~/repos/garlic
git clone https://github.com/justanotherspy/sproot.git ~/repos/sproot
git clone https://gitlab.com/group/project.git ~/code/project
