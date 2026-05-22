#!/usr/bin/env bash
set -euo pipefail

# authenticate the gh CLI using a token from the environment
echo "$GH_TOKEN" | gh auth login --with-token

# install docker, configure the daemon, and run it
curl -fsSL https://get.docker.com | sh
sudo mkdir -p /etc/docker
cat > /etc/docker/daemon.json <<'EOF'
{"storage-driver": "overlay2"}
EOF
sudo dockerd &
