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
| Configure Claude Code (settings, upgrade, CLAUDE.md) | `claude` |
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

### binary_release: tools with non-standard versions and arch strings

Use `version` for tools whose asset names drop the leading `v`, and `arch_map` +
`{arch_alias}` for arch naming that the built-in vars do not cover:

```yaml
# gitleaks: tag is v8.30.1 but the asset is gitleaks_8.30.1_... (no v); arch is x64
- type: binary_release
  name: gitleaks
  repo: gitleaks/gitleaks
  version: "{tag_no_v}"
  asset: "gitleaks_{version}_linux_{x64_arch}.tar.gz"
  install: tar+install

# hadolint: x86_64 on amd64 but arm64 on arm64 (no single built-in var fits)
- type: binary_release
  name: hadolint
  repo: hadolint/hadolint
  arch_map:
    amd64: x86_64
    arm64: arm64
  asset: "hadolint-linux-{arch_alias}"
  install: raw
```

### binary_release: cosign-verified installs

For tools whose install scripts verify a Sigstore keyless signature (e.g. trufflehog),
use the `cosign` block instead of a `cmd` shell-out. Install cosign in an earlier phase:

```yaml
- type: binary_release
  name: trufflehog
  repo: trufflesecurity/trufflehog
  version: "{tag_no_v}"
  asset: "trufflehog_{version}_linux_{arch}.tar.gz"
  checksum_asset: "trufflehog_{version}_checksums.txt"
  install: tar+install
  cosign:
    blob: "trufflehog_{version}_checksums.txt"
    signature: "trufflehog_{version}_checksums.txt.sig"
    certificate: "trufflehog_{version}_checksums.txt.pem"
    certificate_oidc_issuer: "https://token.actions.githubusercontent.com"
    certificate_identity_regexp: "https://github.com/trufflesecurity/trufflehog/.github/workflows/.*"
```

### docker: daemon configuration

Use `daemon_json` to configure the Docker daemon at install time. It is deep-merged
into any existing file. On sprites, include `storage-driver: overlay2` and start dockerd
via a following `sprite_service` phase (no systemd restart happens on a sprite):

```yaml
- type: docker
  daemon_json:
    storage-driver: overlay2
    log-driver: json-file
    log-opts:
      max-size: 10m
      max-file: "3"
    insecure-registries:
      - registry.local:5000

- type: sprite_service
  service: dockerd
  cmd: /usr/bin/sudo      # dockerd needs root; sprite-env runs services as the sprite user
  args:
    - /usr/bin/dockerd
```
