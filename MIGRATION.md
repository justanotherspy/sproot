# Migration Guide

sproot replaces manually maintained bash setup scripts with a declarative YAML config (`sproot.yaml`). This guide covers migration patterns and known module limitations.

## From bash scripts to sproot.yaml

Map common bash idioms to sproot module types:

| Bash pattern | sproot module type |
|---|---|
| `apt-get install <pkg>` | `apt` |
| `pip install` / `uv tool install` | `uv_tool` |
| `curl ... \| tar xz` / GitHub release download | `binary_release` |
| `git config --global ...` | `git_identity` |
| Template a config file from env vars | `file_template` |
| Add a block to `.bashrc` / `.zshrc` | `rc_block` |
| Configure Claude Code settings | `claude_settings` |
| Pull in a dotfiles repo | `git` |
| Install and start Docker | `docker` |
| Set GitHub token for `gh` CLI | `gh_token` |
| Add SSH key to GitHub | `ssh_setup` |
| Anything else | `cmd` |

## Known cmd workarounds

The items below are gaps in existing module types. Until they are addressed, use the `cmd` module type as a workaround.

### apt: post-install symlinks

The `apt` module has no `symlinks` field. Tools like `bat` (installed as `batcat`) and `fd` (installed as `fdfind`) need a symlink to their conventional name.

Workaround:

```yaml
- type: cmd
  name: bat symlink
  run: ln -sf $(which batcat) /usr/local/bin/bat
```

### uv_tool: auto-install uv

The `uv_tool` module does not install `uv` if it is absent. The phase will fail with a "command not found" error.

Workaround: add a `cmd` phase before any `uv_tool` phase:

```yaml
- type: cmd
  name: install uv
  run: curl -LsSf https://astral.sh/uv/install.sh | sh
```

### uv_tool: pkg field for differing package names

The `uv_tool` module installs the tool whose name matches the `tool` field. When the PyPI package name differs from the binary name (e.g. the `garlic` package installs `g`), there is no way to specify the package name separately.

Workaround:

```yaml
- type: cmd
  name: install garlic
  run: uv tool install garlic
```

### binary_release: non-standard arch template variables

The `binary_release` module provides `{arch}` (e.g. `amd64`) and `{os}` template variables. Some tools use non-standard strings in their archive names:

- `gitleaks` uses `x64` instead of `amd64`
- `hadolint` uses `x86_64` instead of `amd64`

There are no `{x64_arch}` or `{x86_64_arch}` template variables yet.

Workaround:

```yaml
- type: cmd
  name: install gitleaks
  run: |
    VER=8.21.2
    curl -sSL "https://github.com/zricethezav/gitleaks/releases/download/v${VER}/gitleaks_${VER}_linux_x64.tar.gz" \
      | tar xz -C /usr/local/bin gitleaks
```

### docker: daemon_json configuration

The `docker` module installs Docker but has no `daemon_json` field for configuring the daemon (e.g. setting `insecure-registries`, `log-driver`, or `registry-mirrors`).

Workaround:

```yaml
- type: cmd
  name: configure docker daemon
  run: |
    mkdir -p /etc/docker
    cat > /etc/docker/daemon.json <<'EOF'
    {
      "log-driver": "json-file",
      "log-opts": { "max-size": "10m", "max-file": "3" }
    }
    EOF
    systemctl restart docker
```
