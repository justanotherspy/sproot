#!/bin/bash
# Entry point for the OpenClaw Sprite Builder web app.
# Registered as the sprite-env managed service "openclaw-builder" (see ../sproot.yaml).
set -euo pipefail
cd "$HOME/openclaw-builder"
exec node server.js
