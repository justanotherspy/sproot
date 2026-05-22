# Migration Guide

sproot replaces manually maintained bash setup scripts with a declarative YAML config (`sproot.yaml`). This guide covers migration patterns.

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

## Common patterns

### apt: tools with different binary names

Some Ubuntu packages install under a different binary name. Use `symlinks` to create the conventional name:

```yaml
- type: apt
  packages:
    - bat      # installs as batcat
    - fd-find  # installs as fdfind
  symlinks:
    - from: /usr/bin/batcat
      to: /usr/local/bin/bat
    - from: /usr/bin/fdfind
      to: /usr/local/bin/fd
```

### uv_tool: tools where PyPI package name differs from binary name

Use the `pkg` field when the PyPI package name differs from the installed binary name:

```yaml
- type: uv_tool
  tools:
    - name: g       # binary name on PATH
      pkg: garlic   # PyPI package to install
```

### binary_release: tools with non-standard arch strings

Use `{x64_arch}` or `{x86_64_arch}` for tools that name their assets differently from Go's `amd64`:

```yaml
# gitleaks uses x64 (not amd64)
- type: binary_release
  name: gitleaks
  repo: zricethezav/gitleaks
  asset: "gitleaks_{version}_linux_{x64_arch}.tar.gz"
  install: tar+install

# hadolint uses x86_64 (not amd64)
- type: binary_release
  name: hadolint
  repo: hadolint/hadolint
  asset: "hadolint-Linux-{x86_64_arch}"
  install: raw
```

### docker: daemon configuration

Use `daemon_json` to configure the Docker daemon at install time:

```yaml
- type: docker
  daemon_json:
    log-driver: json-file
    log-opts:
      max-size: 10m
      max-file: "3"
    insecure-registries:
      - registry.local:5000
```
