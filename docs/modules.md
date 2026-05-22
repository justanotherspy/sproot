# Module Reference

Each phase in `sproot.yaml` is driven by a module type. This document describes all 17 types.

---

## env block (top-level)

The optional `env` block at the top of `sproot.yaml` declares host environment variables to forward into the sprite before `sproot setup` runs. Each entry maps a host variable name to a name inside the sprite.

```yaml
env:
  - from: MY_GH_TOKEN   # variable name on the host
    as: GH_TOKEN        # variable name inside the sprite
    required: true      # fail sproot new if unset on host
```

**Fields:**
- `from`: the environment variable name to read from the host (required)
- `as`: the environment variable name to set in the sprite (required)
- `required`: if `true`, `sproot new` fails early when the host variable is unset or empty (default `false`)

The canonical use is forwarding a GitHub PAT so that `ssh_setup` and `gh_token` can register the sprite's SSH key and authenticate `gh` without embedding tokens in the config repo.

---

## apt

Installs system packages via `apt-get` and optionally creates post-install symlinks.

```yaml
- type: apt
  packages:
    - git
    - curl
    - bat        # installs as batcat on Ubuntu
  symlinks:
    - from: /usr/bin/batcat
      to: /usr/local/bin/bat
```

**Fields:**
- `packages`: list of apt package names
- `symlinks`: optional list of `{from, to}` pairs; `from` is the source path, `to` is the symlink path to create (uses `ln -sf`)

**Idempotency:** checks `dpkg -s <pkg>` for each package and `stat <to>` for each symlink; skips the phase if all are satisfied.

**Platform:** Linux with apt.

---

## uv_tool

Installs Python tools via `uv tool install`. Installs `uv` automatically to `/usr/local/bin` if not already on PATH.

```yaml
- type: uv_tool
  tools:
    - name: ruff
    - name: pyright
    - name: black
    - name: g        # binary name
      pkg: garlic    # PyPI package name (when it differs from the binary name)
```

**Fields:**
- `name`: binary name (used for PATH check and as default package name)
- `pkg`: optional PyPI package name when it differs from the binary name

**Idempotency:** checks that `uv` and each tool binary are on PATH.

---

## go_install

Installs Go tools via `go install`.

```yaml
- type: go_install
  tools:
    - pkg: golang.org/x/tools/cmd/goimports
      version: latest
    - pkg: github.com/golangci/golangci-lint/v2/cmd/golangci-lint
      version: v2.1.6
```

**Fields:**
- `pkg`: full Go module path for the tool
- `version`: `latest` or a full semver like `v1.2.3`

**Idempotency:** for semver versions, checks that the binary is on PATH and `go version -m` reports the expected module path. `latest` always re-runs.

**Requires:** `go` on PATH.

---

## cargo_install

Installs Rust tools via `cargo install`.

```yaml
- type: cargo_install
  tools:
    - name: ripgrep
    - name: cargo-edit
      version: "0.12.2"
      locked: true
    - name: sccache
      features:
        - dist-client
```

**Fields:**
- `name`: crate name
- `version`: optional; omit for latest
- `locked`: optional; passes `--locked`
- `features`: optional list of Cargo features to enable

**Idempotency:** checks `cargo install --list` for `<name> v<version>`.

**Requires:** `cargo` on PATH.

---

## binary_release

Downloads and installs a binary from a GitHub release.

```yaml
- type: binary_release
  name: cosign
  repo: sigstore/cosign
  asset: "cosign_{version}_{arch}.deb"
  install: dpkg
  checksum: ""         # optional: sha256 hex of the downloaded asset
  checksum_asset: ""   # optional: asset name template for a checksums file
```

**Fields:**
- `name`: tool name (used for PATH check or dpkg package name)
- `repo`: `owner/repo` on GitHub
- `asset`: asset filename template (see template variables below)
- `install`: one of `dpkg`, `tar+install`, or `raw`
- `checksum`: optional sha256 hex string; verified against the downloaded asset before install
- `checksum_asset`: optional asset name template (e.g. `{name}_{version}_checksums.txt`) for a goreleaser-style checksums file; sproot downloads it, finds the matching line, and verifies

**Asset template variables:**
- `{version}`: latest release tag (e.g. `v2.4.1`)
- `{arch}`: `amd64` or `arm64`
- `{goos}`: `linux` or `darwin`
- `{dpkg_arch}`: Debian arch name (`amd64`, `arm64`)
- `{x64_arch}`: x64-style arch (`x64` on amd64, `arm64` on arm64)
- `{x86_64_arch}`: x86_64-style arch (`x86_64` on amd64, `aarch64` on arm64)

**Install methods:**
- `dpkg`: runs `dpkg -i <file>`; idempotency via `dpkg -s <name>`
- `tar+install`: extracts tarball, copies the single executable to `/usr/local/bin/<name>`
- `raw`: marks the downloaded file executable and copies to `/usr/local/bin/<name>`

**Idempotency:** for `dpkg` checks `dpkg -s <name>`; for others checks binary on PATH.

---

## corepack

Enables corepack and pre-activates the listed package managers.

```yaml
- type: corepack
  managers: [pnpm, yarn]
```

**Fields:**
- `managers`: list of package manager names to prepare (e.g. `pnpm`, `yarn`)

**Idempotency:** checks that each listed manager binary is on PATH.

**Requires:** `corepack` on PATH (ships with Node.js 16+).

---

## rust_components

Pins the stable Rust toolchain and installs the listed components.

```yaml
- type: rust_components
  components: [clippy, rustfmt, rust-analyzer]
```

**Fields:**
- `components`: list of rustup component names to install

Also sets `rustup default stable`.

**Idempotency:** checks `rustup component list --installed` for each component.

**Requires:** `rustup` on PATH.

---

## docker

Installs Docker via the official install script. Optionally writes `/etc/docker/daemon.json`.

```yaml
- type: docker

# with daemon configuration:
- type: docker
  daemon_json:
    log-driver: json-file
    log-opts:
      max-size: 10m
      max-file: "3"
    insecure-registries:
      - registry.local:5000
```

**Fields:**
- `daemon_json`: optional map written to `/etc/docker/daemon.json` after install; docker is restarted to apply it

**Idempotency:** checks `docker --version` exits 0; if `daemon_json` is set, also checks that `/etc/docker/daemon.json` exists.

**Platform:** Linux. Requires root or sudo.

---

## sprite_service

Registers a long-running service with the sprite-env daemon.

```yaml
- type: sprite_service
  service: dockerd
  cmd: /usr/bin/dockerd
  args:
    - --host=unix:///var/run/docker.sock
  http_port: 2375        # optional: port sprite-env monitors for health
  needs: [networking]    # optional: services that must start first
```

**Fields:**
- `service`: service name (used as path key in the API)
- `cmd`: executable path
- `args`: optional command arguments
- `http_port`: optional port number; when set, sprite-env monitors it for health checks
- `needs`: optional list of service names that must be running before this service starts

**Idempotency:** checks `sprite-env curl /v1/services/<name>` exits 0.

**Platform:** sprite-env only (sprite.dev sprites).

---

## git_identity

Configures global git identity and optional additional git settings.

```yaml
- type: git_identity
  config:
    pull.rebase: "true"
    push.autoSetupRemote: "true"
    core.editor: vim
```

Always sets `user.name`, `user.email`, and `init.defaultBranch` from the top-level `identity` block. The optional `config` map lets the config repo apply any additional git settings without sproot prescribing them.

When `~/.ssh/id_ed25519.pub` exists, also sets SSH commit signing (`gpg.format`, `user.signingkey`, `commit.gpgsign`, `tag.gpgsign`, `gpg.ssh.allowedSignersFile`).

**Idempotency:** checks identity fields, each key in `config`, and (when the pub key exists) `gpg.format`. Treats the sprite placeholder email `noreply@sprites.dev` as "not configured."

---

## ssh_setup

Generates a fresh ed25519 keypair and registers it with GitHub.

```yaml
- type: ssh_setup
```

- Generates `~/.ssh/id_ed25519` and `~/.ssh/id_ed25519.pub` if absent
- Sets permissions on the private key (0600)
- Registers the public key with GitHub as both an authentication key and a signing key using `GH_TOKEN` (forwarded via the `env` block in `sproot.yaml` or via `gh_token_env` in `~/.sproot/config.yaml`)
- Runs `ssh-keyscan -H github.com` and appends to `~/.ssh/known_hosts`
- Appends the user's key to `~/.ssh/allowed_signers` with the `namespaces="git"` constraint

The GitHub key IDs are logged for use by `sproot destroy` when cleaning up the sprite account (Phase 5). If `GH_TOKEN` is not set, key generation and local setup proceed but GitHub registration is skipped with a warning.

**Idempotency:** checks that `~/.ssh/id_ed25519` exists and `~/.ssh/known_hosts` contains the github.com host key.

**Required GitHub token scopes:** `admin:public_key`, `admin:ssh_signing_key`

---

## gh_token

Authenticates `gh` (GitHub CLI) by persisting credentials from `GH_TOKEN`.

```yaml
- type: gh_token
```

Reads `GH_TOKEN` from the environment (forwarded via the `env` block in `sproot.yaml` or via `gh_token_env` in `~/.sproot/config.yaml`) and pipes it to `gh auth login --hostname github.com --git-protocol ssh --with-token`. After this runs, `gh` works from stored credentials in `~/.config/gh/hosts.yml` without needing `GH_TOKEN` set in future sessions.

**Idempotency:** checks `gh auth status -h github.com` exits 0 and the logged-in user matches `identity.gh_username`.

**Requires:** `GH_TOKEN` set in the sprite environment. sproot itself imposes no minimum scope here; match whatever you want `gh` to do inside the sprite (typically `repo`, plus `read:org` for cross-organization work).

---

## file_template

Copies or renders a file from the config repo to a destination path.

```yaml
- type: file_template
  src: files/gitconfig
  dest: ~/.gitconfig
  mode: "0644"
  template: true   # set true to render Go template variables
```

**Fields:**
- `src`: path relative to the config repo root
- `dest`: destination path (`~` is expanded)
- `mode`: optional file permissions (default: `0644`)
- `template`: optional; when `true`, executes `src` as a Go template with `ctx.Identity` as data. When `false` (the default), the file is copied as-is.

**Template data fields:** `GitUserName`, `GitUserEmail`, `GitDefaultBranch`, `GHUsername`

**Idempotency:** checks that destination content matches the rendered source.

---

## rc_block

Injects a managed shell block into `.bashrc` and `.zshrc`.

```yaml
- type: rc_block
  src: rc.sh
```

**Fields:**
- `src`: path relative to the config repo root

Wraps the source content in sentinel comments:
```
# BEGIN SPROOT MANAGED BLOCK
<contents of src>
# END SPROOT MANAGED BLOCK
```

Both `.bashrc` and `.zshrc` are updated. On re-run, the existing block is replaced (not duplicated).

**Idempotency:** checks that the sentinel block is present and the content hash matches the source file.

---

## repo_clone

Clones repositories into configured destinations.

```yaml
- type: repo_clone
  base_dir: ~/repos
  repos:
    - justanotherspy/sproot          # short form: SSH clone into base_dir/<repo>
    - justanotherspy/sprite
    - url: https://github.com/org/project.git   # long form: explicit URL
      dest: ~/project                            # optional explicit destination
```

**Short form** (`owner/repo` string): clones via SSH (`git@github.com:owner/repo.git`) into `<base_dir>/<repo>`. Requires `base_dir`.

**Long form** (`{url, dest}` map): clones the given URL. `dest` is optional; when omitted, defaults to `~/<repo-name>` (last URL path component, minus `.git`).

**Fields:**
- `base_dir`: base directory for short-form repos (`~` is expanded)
- `repos`: list of short-form strings or `{url, dest}` maps

**Idempotency:** skips repos where the destination directory already contains `.git`.

---

## claude_settings

Deep-merges settings into `~/.claude/settings.json`.

```yaml
- type: claude_settings
  settings:
    theme: dark
    autoApprove: true
    env:
      ANTHROPIC_SMALL_FAST_MODEL: claude-haiku-4-5-20251001
```

**Fields:**
- `settings`: arbitrary map of keys and values to merge into the settings file

Existing keys not listed in `settings` are preserved. Nested maps are merged recursively.

**Idempotency:** checks that all specified keys already match target values.

---

## npm

Runs `npm install` in a project directory.

```yaml
- type: npm
  dir: ~/my-project
```

**Fields:**
- `dir`: directory containing `package.json` (`~` is expanded)

**Idempotency:** skips when `node_modules` already exists in `dir`.

---

## cmd

Runs an arbitrary shell command.

```yaml
- type: cmd
  run: "curl -fsSL https://example.com/install.sh | sh"
  check: "which mytool"
  name: "install-mytool"   # optional; shown as cmd(install-mytool) in status output
```

**Fields:**
- `run`: shell command to execute (passed to `sh -c`)
- `check`: optional shell command; if it exits 0, the phase is skipped
- `name`: optional display name; when set, the phase appears as `cmd(name)` in status output. Useful when multiple `cmd` phases are present.

**Idempotency:** if `check` is provided, skips when it exits 0. Without `check`, always runs.

Use this as an escape hatch for one-off operations not covered by other module types.
