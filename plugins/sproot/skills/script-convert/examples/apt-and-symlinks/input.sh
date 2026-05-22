#!/usr/bin/env bash
set -euo pipefail

# system packages
sudo apt-get update
sudo apt-get install -y git jq ripgrep bat fd-find

# expose Ubuntu's renamed binaries under their expected names
ln -sf /usr/bin/batcat /usr/local/bin/bat
ln -sf /usr/bin/fdfind /usr/local/bin/fd
