#!/usr/bin/env bash
set -euo pipefail

# a static config file
mkdir -p ~/.config/nvim
cat > ~/.config/nvim/init.lua <<'EOF'
vim.opt.number = true
vim.opt.expandtab = true
EOF

# shell init across bash and zsh
cat >> ~/.bashrc <<'EOF'
export EDITOR=nvim
alias ll='ls -alF'
EOF
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
