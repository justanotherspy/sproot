#!/usr/bin/env bash
set -euo pipefail

# install cosign from its GitHub releases (.deb asset; the asset name drops the leading v)
curl -fsSL -o /tmp/cosign.deb \
  "https://github.com/sigstore/cosign/releases/download/v2.4.1/cosign_2.4.1_amd64.deb"
sudo dpkg -i /tmp/cosign.deb
rm -f /tmp/cosign.deb
