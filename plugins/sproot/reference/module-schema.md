# sproot module schema (condensed)

Authoritative field reference for every `sproot.yaml` module type. The canonical source is
`internal/config/schema.go` and `docs/modules.md` in this repo; this file is a compact mirror
for skills. Keep it in sync when modules change.

**All phase configs are flat**: a module's fields sit directly under the list item next to
`type:`. There is no nesting like `corepack: { managers: ... }`; it is `type: corepack` then
`managers: [...]` at the same level.

## Top-level `sproot.yaml`

```yaml
schema_version: 1            # required, must be 1
identity:                    # required block (all four fields required by validate)
  git_user_name: "Name"
  git_user_email: "email@example.com"
  git_default_branch: main
  gh_username: github-handle
env:                         # optional; forwards host env vars into the sprite
  - from: HOST_VAR_NAME      # required; var name on the host
    as: SPRITE_VAR_NAME      # required; var name inside the sprite
    required: true           # optional; fail `sproot new` if unset on host
checkpoint_after_setup: false  # optional
phases: [ ... ]              # EITHER phases ...
targets:                     # ... OR targets (not both)
  default:
    phases: [ ... ]
  web:
    extends: default         # optional inheritance; parent phases run first
    phases: [ ... ]
```

`phases` and `targets` are mutually exclusive; exactly one must be non-empty. A target may set
`extends` (a parent target name) and/or `phases`.

## Modules

Required fields are marked `(req)`. Anything else is optional.

### apt: install system packages
```yaml
- type: apt
  packages: [git, jq]        # at least one of packages/symlinks required
  symlinks:                  # optional post-install symlinks (~ expanded, parent dir created)
    - from: /usr/bin/batcat
      to: ~/.local/bin/bat
```
Ubuntu name quirks: `bat` installs as `batcat`, `fd-find` as `fdfind`: add `symlinks` to expose the expected name. Drop any `sudo` prefix (the module escalates itself).

### uv_tool: Python CLI tools via `uv tool install`
```yaml
- type: uv_tool
  tools:                     # (req) non-empty
    - name: ruff             # (req) binary name (also default package name)
    - name: g
      pkg: garlic            # optional PyPI package when it differs from name
```

### go_install: Go tools via `go install pkg@version`
```yaml
- type: go_install
  tools:                     # (req) non-empty
    - pkg: golang.org/x/tools/cmd/goimports   # (req)
      version: latest        # optional ("latest" or vX.Y.Z); "latest" always re-runs
```

### cargo_install: Rust crates via `cargo install`
```yaml
- type: cargo_install
  tools:                     # (req) non-empty
    - name: ripgrep          # (req) crate name
      bin: rg                # optional, binary name when it differs from the crate
      version: "0.9.72"      # optional
      locked: true           # optional, passes --locked
      features: [foo]        # optional
```

### binary_release: install a GitHub release asset
```yaml
- type: binary_release
  name: cosign               # (req) display + dpkg package name; used in idempotency check
  repo: sigstore/cosign      # (req) owner/repo
  asset: "cosign_{version}_{arch}.deb"  # (req) asset name template
  install: dpkg              # (req) one of: dpkg | tar+install | raw
  version: "{tag_no_v}"      # optional template for the {version} token (default: raw tag)
  arch_map: {amd64: x86_64}  # optional; REQUIRED if asset/checksum/cosign uses {arch_alias}
  checksum: "<sha256hex>"    # optional
  checksum_asset: "checksums.txt"  # optional goreleaser-style checksums file template
  cosign:                    # optional keyless verify (needs cosign on PATH from an earlier phase)
    certificate_identity_regexp: ".*"          # (req within cosign)
    certificate_oidc_issuer: "https://token.actions.githubusercontent.com"  # (req)
    blob: "checksums.txt"                       # (req)
    signature: "checksums.txt.sig"              # (req)
    certificate: "checksums.txt.pem"            # (req)
```
Asset template vars: `{version}` `{tag}` `{tag_no_v}` `{arch}` `{goos}` `{dpkg_arch}` `{x64_arch}` `{x86_64_arch}` `{arch_alias}`.

### corepack: enable corepack package managers
```yaml
- type: corepack
  managers: [pnpm, yarn]     # (req) non-empty
```

### rust_components: pin stable + add rustup components
```yaml
- type: rust_components
  components: [clippy, rustfmt, rust-analyzer]  # (req) non-empty
```

### docker: install docker-ce, optional daemon.json deep-merge
```yaml
- type: docker
  daemon_json:               # optional; deep-merged into /etc/docker/daemon.json
    storage-driver: overlay2
```

### sprite_service: register a sprite-env managed service
```yaml
- type: sprite_service
  service: dockerd           # service name
  cmd: /usr/bin/dockerd      # (req) executable path
  args: [--flag]             # optional
  http_port: 8080            # optional
  needs: [dockerd]           # optional dependencies
```

### git_identity: apply identity + extra git config
```yaml
- type: git_identity
  config:                    # optional extra key/values; identity comes from top-level block
    pull.rebase: "true"
    core.editor: vim
```

### ssh_setup: generate + register SSH key (no fields)
```yaml
- type: ssh_setup
```
Needs `GH_TOKEN` in the sprite (PAT with `admin:public_key`, `admin:ssh_signing_key`) via the `env` block.

### gh_token: authenticate `gh` from GH_TOKEN (no fields)
```yaml
- type: gh_token
```
Needs `GH_TOKEN` in the sprite via the `env` block.

### file_template: copy/render a config-repo file to a destination
```yaml
- type: file_template
  src: files/statusline.py   # (req) config-repo-relative path
  dest: ~/.claude/statusline.py  # (req) destination (~ expanded)
  mode: "0755"               # optional octal string
  template: false            # optional; render as Go template against identity
```

### rc_block: managed block appended to ~/.bashrc and ~/.zshrc
```yaml
- type: rc_block
  src: files/rc_additions.sh # (req) config-repo-relative path
```

### repo_clone: clone repos
```yaml
- type: repo_clone
  base_dir: ~/repos          # (req) base for short-form entries
  repos:                     # (req) non-empty
    - justanotherspy/garlic  # short form: cloned via SSH into base_dir/<repo>
    - url: https://github.com/org/project.git  # long form
      dest: ~/project        # optional (defaults to ~/<repo-name>)
```

### npm: run `npm install` in a directory
```yaml
- type: npm
  dir: ~/my-project          # (req)
```

### shell_completion: generate + install shell completions
```yaml
- type: shell_completion
  completions:               # (req) non-empty
    - command: sproot        # (req)
      shells: [bash, zsh, fish]  # (req) non-empty; each one of bash|zsh|fish
    - command: weird
      shells: [bash]
      gen: "{command} --completion {shell}"  # optional; default "{command} completion {shell}"
```
Installs to per-user dirs (bash: `~/.local/share/bash-completion/completions/`, zsh: `~/.zfunc/`, fish: `~/.config/fish/completions/`) and adds a managed zsh `fpath`+`compinit` block to `~/.zshrc` when any entry targets zsh. Order after whatever installs the commands.

### claude: configure Claude Code
```yaml
- type: claude               # set at least one of settings/upgrade/claude_md
  settings:                  # optional, deep-merged into ~/.claude/settings.json
    theme: dark
  upgrade: true              # optional, runs `claude upgrade`
  claude_md: files/CLAUDE.md # optional managed CLAUDE.md block (config-repo path)
  template: false            # optional; render claude_md as Go template
```

### cmd: arbitrary command (the escape hatch)
```yaml
- type: cmd
  run: "curl -fsSL https://example.com/install.sh | sh"  # (req)
  check: "command -v example"  # optional; exits 0 -> skip run (idempotency)
  name: install-example      # optional display name
```

### nix: install Determinate Nix, run nix-daemon, install packages
```yaml
- type: nix
  packages:                  # optional
    - hello                  # short form: nixpkgs#hello, symlinked as "hello"
    - name: ripgrep          # long form (needs name or flake)
      bin: rg                # optional; binary name when it differs from the package
    - name: nixfmt
      flake: "nixpkgs#nixfmt-rfc-style"  # optional; default nixpkgs#<name>
      bin: nixfmt
  setup_script: files/nix-setup.sh  # optional; config-repo path, run with the nix profile sourced
  daemon_service: true       # optional, default true; register nix-daemon as a sprite service
```
Symlinks the `nix` CLI and each package binary into `~/.local/bin` (on the base PATH for `sprite exec`/services) and sources the nix profile from the login shells. nix-daemon runs as a sprite service (sprites have no systemd), like `docker` + `sprite_service`. Needs root (the installer self-escalates) and network access to `install.determinate.systems` + `*.nixos.org`.
