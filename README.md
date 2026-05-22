# sproot

sproot bootstraps [sprite.dev](https://sprite.dev) sprites from a user-owned config repo. It replaces bash-based setup scripts with a single, versioned Go binary driven by a `sproot.yaml` file you control.

## How it works

```
sproot new my-sprite
```

1. Reads `~/.sproot/config.yaml` for your config repo URL and token env var names
2. Resolves the API and GitHub tokens from the env vars named in `~/.sproot/config.yaml` (`token_env` and `gh_token_env`)
3. Creates a new sprite via the sprites-go SDK
4. Injects the sproot binary into the sprite at `/usr/local/bin/sproot`
5. Runs `sproot setup` inside the sprite with `GH_TOKEN` forwarded, which clones your config repo and executes each phase
6. The `ssh_setup` phase generates a fresh ed25519 keypair, registers it with GitHub using `GH_TOKEN`, and records the key IDs for cleanup

Each phase is idempotent. Re-running setup is safe.

## Config repo

Your config repo holds a `sproot.yaml` that lists the phases to run:

```yaml
schema_version: 1

identity:
  git_user_name: "Your Name"
  git_user_email: "you@example.com"
  git_default_branch: main
  gh_username: yourname

checkpoint_after_setup: true   # optional; creates a checkpoint after setup completes

phases:
  - type: apt
    packages: [shellcheck, jq]
  - type: uv_tool
    tools: [{name: ruff}]
  - type: file_template
    src: files/statusline.py
    dest: ~/.claude/statusline.py
    mode: "0755"
  - type: rc_block
    src: files/rc_additions.sh
  - type: repo_clone
    base_dir: ~/repos
    repos:
      - yourname/yourrepo
```

See [docs/modules.md](docs/modules.md) for all module types.

### Multi-target configs

A single `sproot.yaml` can define named targets so one config repo produces several sprite flavors. Use `extends` to inherit phases from another target:

```yaml
targets:
  base:
    phases:
      - type: apt
        packages: [curl, git]
  web:
    extends: base
    phases:
      - type: apt
        packages: [nginx]
```

```sh
sproot new my-web-sprite --target web
sproot push --target web
```

A flat `phases:` list (no `targets:`) continues to work unchanged.

### Local config source

For development or generated configs, you can point sproot at a local directory instead of a git repo:

```yaml
# ~/.sproot/config.yaml
config_source: local
config_local_path: ~/my-sprite-config
token_env: SPRITES_TOKEN
```

Or pass it inline:

```sh
sproot new my-sprite --local-config ~/my-sprite-config
```

sproot uploads the directory to the sprite and runs setup without a git clone.

## Host config

```
~/.sproot/
└── config           # YAML: config_repo, token_env, gh_token_env
```

`~/.sproot/config.yaml` format:

```yaml
config_repo: git@github.com:yourname/sprite.git
config_ref: main
config_path: ""             # optional; path to sproot.yaml within the config repo
token_env: SPRITES_TOKEN     # name of env var holding your sprites API token
gh_token_env: GITHUB_TOKEN  # name of env var holding your GitHub PAT
default_org: ""
# config_source: local      # set to "local" to use a directory instead of a git repo
# config_local_path: ~/my-sprite-config
```

The config stores environment variable **names**, not token values. Tokens stay in your shell environment (e.g. exported from your password manager or `.profile`). Each sprite generates its own SSH keypair; `sproot destroy` removes it from GitHub automatically.

Initialize with:

```
sproot config init
```

## Commands

| Command | Where | Description |
|---------|-------|-------------|
| `sproot new <name>` | host | Create and provision a sprite (`--skip-console` to skip opening a shell, `--skip-verify` to skip the built-in end-of-run verification) |
| `sproot destroy <name>` | host | Destroy a sprite and remove its GitHub SSH keys (`--force` skips confirmation) |
| `sproot status <name>` | host | Show setup state (exec into sprite); `--host` reads state without exec |
| `sproot console <name>` | host | Open an interactive shell in a sprite |
| `sproot exec <name> <cmd> [args...]` | host | Run a one-off command in a sprite and stream output (`--env KEY=val,K2=v2`) |
| `sproot list` | host | List sproot-managed sprites (`--all` shows every sprite, `--prefix` filters by name, `--watch` refreshes live) |
| `sproot upgrade <name>` | host | Upgrade the sprite CLI inside a sprite |
| `sproot checkpoint <name>` | host | Create a checkpoint (`--comment` adds a label) |
| `sproot checkpoints <name>` | host | List checkpoints (`--include-auto` shows auto checkpoints) |
| `sproot restore <name> <id>` | host | Restore a sprite from a checkpoint |
| `sproot push` | host | Re-run setup on all sproot-managed sprites (`--name` for one, `--target` to select a target, `--no-checkpoint` to skip pre-push checkpoint, `--skip-verify` to skip end-of-run checks) |
| `sproot outdated` | host | Show which sproot-managed sprites have a stale config SHA |
| `sproot config init` | host | Interactive `~/.sproot/config.yaml` setup (`--non-interactive` for scripting) |
| `sproot config validate` | host | Validate `~/.sproot/config.yaml` only |
| `sproot validate [--path PATH]` | host | Validate a sproot.yaml (also validates `~/.sproot/config.yaml`); missing file is a warning unless `--strict` is passed |
| `sproot setup` | sprite | Clone config repo and run phases |
| `sproot setup --status` | sprite | Print phase state table |

## Quickstart

**1. Initialize host config:**

```sh
sproot config init
```

Edit `~/.sproot/config.yaml` and fill in your config repo URL and token env var names.

**2. Validate your config:**

```sh
sproot config validate
```

**3. Create a sprite:**

```sh
sproot new my-sprite
```

This creates the sprite, injects the sproot binary, and runs `sproot setup` inside it.

**4. Check setup status:**

```sh
sproot status my-sprite
```

**5. Tear down:**

```sh
sproot destroy my-sprite
```

## Installation

Binaries are available on the [releases page](https://github.com/justanotherspy/sproot/releases).

**One-line install (Linux and macOS):**

```sh
curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh
```

The installer detects your OS and architecture, downloads the correct archive, verifies the SHA256 checksum, and places the binary in `/usr/local/bin` (if writable) or `~/.local/bin`.

To install a specific version, set `SPROOT_VERSION`:

```sh
SPROOT_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh
```

**From source:**

```sh
git clone https://github.com/justanotherspy/sproot
cd sproot
make install
```

Requires Go 1.25+.

## Verifying release signatures

Releases are signed with [cosign](https://github.com/sigstore/cosign) keyless signing via Sigstore. To verify:

```sh
cosign verify-blob \
  --bundle sproot_v0.1.0_checksums.txt.sigstore.json \
  sproot_v0.1.0_checksums.txt
```

A successful verify confirms the checksum file was signed by the GitHub Actions release workflow for this repository.

## Development

```sh
make build              # build ./sproot
make test               # run tests
make check              # vet + test + lint
make lint               # golangci-lint
make release-check      # validate .goreleaser.yaml syntax
make release-dry-run    # local snapshot build (requires goreleaser)
make e2e                # end-to-end tests against real sprites (requires SPRITES_TOKEN)
```

`make e2e` creates real sprites using `testdata/integration` configs, verifies labels, tests push, and destroys everything. All sprites are destroyed in cleanup even if a step fails.

Requires Go 1.25+.

## License

MIT
