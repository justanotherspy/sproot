# sproot: plan

A Go CLI for bootstrapping sprite.dev sprites from a user-owned config repo.
Replaces the current bash-based `sprite` repo with a generic, reusable tool.

## Two-repo model

- **`justanotherspy/sproot`** (new): the CLI source + release binaries. Generic, no personal config.
- **`justanotherspy/sprite`** (existing): becomes a reference *config repo*. Holds a `sproot.yaml` plus accompanying files (statusline.py, claude settings, etc). Anyone can fork it or write their own.

## Host file layout

```
~/.sproot/
├── config           # YAML: token, config_repo_url, private_key_path, defaults
└── private/
    └── id_ed25519   # user's sprite SSH key (loaded from password manager)
```

## End-to-end flow

```
sproot new my-sprite
  ├─ reads ~/.sproot/config
  ├─ sprites.New(token).CreateSprite(ctx, "my-sprite", nil)
  ├─ sprite.Filesystem().WriteFile(".ssh/id_ed25519", keyBytes, 0600)
  ├─ sprite.Filesystem().WriteFile("/usr/local/bin/sproot", binaryBytes, 0755)
  ├─ sprite.Command("sproot", "setup", "--config-repo", url).Run()   // streams output
  └─ in-sprite: clone config repo, read sproot.yaml, run phases, verify
```

## Phases (for execution)

Each phase is a discrete deliverable. Phases are roughly in dependency order. Tests are required inline per phase, not deferred.

---

### Phase 0: Scaffold `justanotherspy/sproot` (DONE)

**What was built:**
- `go.mod` (Go 1.23), cobra for command routing, Makefile with build/test/check/lint/tidy targets
- Full directory layout: `cmd/sproot/`, `internal/config/`, `internal/phase/`, `internal/phase/modules/`, `internal/host/`, `internal/sprite/`, `pkg/log/`
- `.github/workflows/ci.yml`: build-and-test + lint jobs on every push
- `cmd/sproot/main.go` wired to cobra; `internal/host/` and `internal/sprite/` are empty stubs

---

### Phase 1: Config schema (DONE)

**What was built:**
- `internal/config/schema.go`: `SprootConfig`, `HostConfig`, `Identity`, `PhaseConfig` and all typed phase config structs with `yaml:""` tags. `PhaseConfig` uses a two-pass custom unmarshaler: first reads `type`, then decodes into the matching concrete struct pointer.
- `internal/config/load.go`: `LoadSprootConfig` and `LoadHostConfig` with `~` expansion
- `internal/config/validate.go`: `ValidateSprootConfig` and `ValidateHostConfig` with field-named error messages; detects unknown module types
- `internal/config/testdata/sproot.yaml` + `testdata/host_config.yaml`
- 35 unit tests covering happy path and every validation failure

**`sproot.yaml`** shape:
```yaml
schema_version: 1

identity:
  git_user_name: "Daniel Schwartz"
  git_user_email: "danielschwar@gmail.com"
  git_default_branch: main
  gh_username: justanotherspy

phases:
  - type: apt
    packages: [shellcheck, jq, ...]
  - type: uv_tool
    tools: [{name: ruff}, {name: pre-commit}]
  - type: binary_release
    name: cosign
    repo: sigstore/cosign
    asset: "cosign_{version}_{arch}.deb"
    install: dpkg
  - type: go_install
    tools: [{pkg: "github.com/owner/tool", version: latest}]
  - type: cargo_install
    tools: [{name: ripgrep}]
  - type: file_template
    src: files/statusline.py
    dest: ~/.claude/statusline.py
    mode: "0755"
  - type: rc_block
    src: files/rc_additions.sh
  - type: repo_clone
    base_dir: ~/repos
    repos:
      - justanotherspy/garlic
      - justanotherspy/poker
```

**`~/.sproot/config`** shape:
```yaml
token: <sprite.dev API token>
config_repo: git@github.com:justanotherspy/sprite.git
config_ref: main
private_key: ~/.sproot/private/id_ed25519
default_org: ""
```

Note: `token` was not in the original schema. It must be added before Phase 5 work begins (update `HostConfig` in `schema.go` and `validate.go`).

---

### Phase 2: Phase engine (DONE)

**What was built:**
- `internal/phase/phase.go`: `Phase` interface (`Type`, `Name`, `ShouldRun`, `Run`, `Verify`) and `Context` struct (config repo path, identity, logger, dry-run, force flags)
- `internal/phase/runner.go`: orchestrates the phase list; tracks did-work / skipped / failed buckets; supports `--only` (by name or index) and `--force`
- `internal/phase/state.go`: reads/writes `~/.config/sproot/state.json`; records per-phase run time, status, and error
- `internal/phase/registry.go`: `Register` + `Build` with `init()`-based module registration
- `pkg/log/log.go`: `+`/`-`/`!`/`x` visual conventions with color support
- Full test coverage: runner tests with dummy phases (pass, fail, skip, force), state round-trip tests, registry tests

---

### Phase 3: Phase module implementations (DONE)

**What was built:**
- All 17 module files in `internal/phase/modules/`, each registered via `init()`:
  - `apt` — install apt packages
  - `uv_tool` — install via `uv tool install`
  - `go_install` — `go install pkg@version`; idempotency via `go version -m $(which <binary>)`
  - `cargo_install` — `cargo install`; supports optional `version`, `features`, `locked`; idempotency via `cargo install --list`
  - `binary_release` — GitHub releases with asset templating (`{version}`, `{arch}`, `{goos}`, `{dpkg_arch}`) and install methods (`dpkg`, `tar+install`, `raw`)
  - `corepack` — enable + pre-activate pnpm and yarn
  - `rust_components` — pin stable, install clippy/rustfmt/rust-analyzer via `rustup`
  - `docker` — docker-ce install + `/etc/docker/daemon.json`
  - `sprite_service` — register a `sprite-env` service (dockerd today, extensible)
  - `git_identity` — user.name, user.email, default branch, aliases, signing config
  - `ssh_setup` — chmod key, populate known_hosts, derive pubkey via `ssh-keygen -y -f`, write `allowed_signers`
  - `gh_token` — export `GH_TOKEN` from sprite app token; verify `gh auth status`
  - `file_template` — copy file from config repo to dest, with optional Go-template execution against identity
  - `rc_block` — write sentinel-delimited block to `.bashrc` and `.zshrc`; strips legacy blocks
  - `repo_clone` — clone a list of `git@github.com:owner/repo` repos into a base dir
  - `claude_settings` — deep-merge a JSON object into `~/.claude/settings.json`
  - `cmd` — escape hatch: arbitrary command with optional idempotency-check command
- `exec.go` — shared `runCmd` helper that streams stdout+stderr line-by-line via `pkg/log`
- `modules.go` — blank-import of all module packages to trigger `init()` registration
- Unit tests for each module, plus `integration_test.go` covering all 17 types under `--dry-run`
- `docs/modules.md` — full module reference with YAML examples

---

### Phase 4: `sproot setup` (in-sprite command)

The command that runs inside the sprite.

**Deliverables:**
- `cmd/sproot/setup.go`:
  - Flags: `--config-repo`, `--ref`, `--only`, `--force`, `--status`, `--dry-run`
  - Clones `--config-repo` to `~/.sproot/config-repo/` (or pulls if present)
  - Loads `sproot.yaml`, validates, runs phases
  - Returns non-zero on failure
- A built-in `verify` phase that runs last: checks commands on PATH, file modes, rc block content, gh auth, ssh to github.com. Same coverage as the current `_lib_verify.sh`.

**Acceptance:** invoked manually inside a fresh sprite with a known-good config, produces a working environment matching current `setup.sh` output. `sproot setup --status` dumps the state file as a table.

---

### Phase 5: Host CLI commands

The commands the user runs on their laptop. All sprite interaction uses the `github.com/superfly/sprites-go` SDK directly, not the `sprite` CLI.

**sprites-go integration:**
- `sprites.New(token)` constructs the client from `~/.sproot/config`'s `token` field
- `client.CreateSprite(ctx, name, *SpriteConfig)` for sprite creation (config allows optional `ram_mb`, `cpus`, `region`)
- `sprite.Filesystem().WriteFile(path, data, perm)` injects the SSH key and sproot binary before running setup
- `sprite.Command("sproot", "setup", ...).Run()` streams setup output with `Stdout`/`Stderr` wired to `os.Stdout`/`os.Stderr`
- `client.DestroySprite(ctx, name)` for destroy
- `client.GetSprite(ctx, name)` for status queries

**Deliverables:**
- `internal/host/`: add `token` field to `HostConfig` (and update `schema.go` + `validate.go`)
- `sproot new <name>`:
  - Reads `~/.sproot/config`
  - Creates sprite via `client.CreateSprite`
  - Reads `private_key` from disk; injects it at `.ssh/id_ed25519` via `sprite.Filesystem().WriteFile`
  - Reads the running sproot binary (`os.Executable()`); injects it at `/usr/local/bin/sproot`
  - Runs `sproot setup --config-repo <url>` via `sprite.Command`, streaming output live
  - Returns the sprite's exit code
- `sproot destroy <name>` — calls `client.DestroySprite`. No GitHub-side cleanup needed under the one-key model.
- `sproot status <name>` — runs `sproot setup --status` via `sprite.Command`, prints the table
- `sproot config init` — writes a skeleton `~/.sproot/config`
- `sproot config validate` — validates `~/.sproot/config` and optionally fetches and validates the config repo's `sproot.yaml`
- Optional `sproot new` flags: `--ram-mb`, `--cpus`, `--region` forwarded into `SpriteConfig`

**Acceptance:** `sproot new my-sprite` produces a working sprite end-to-end against `justanotherspy/sprite` as the config repo. Opening a console on the sprite lands in a usable shell.

---

### Phase 6: Convert `justanotherspy/sprite` into a config repo

Migrate the existing bash repo into a reference sproot config.

**Deliverables:**
- `sproot.yaml`: full translation of every current `setup.sh` phase into module invocations
- `files/`:
  - `statusline.py` (already exists; move here)
  - `ps1.sh` (extract from current `phase_ps1`)
  - `rc_additions.sh` (extract from current `RC_BLOCK`)
  - `gitignore_global` (extract from current `phase_gitignore_global`)
  - `pre-commit-config.template.yaml` (extract from current `phase_pre_commit_template`)
  - `claude-settings.json` (the managed-keys subset)
- `README.md`: explain that the repo is now a config example, point at `justanotherspy/sproot` for the tool
- Archive or delete `setup.sh`, `post.sh`, `pre.sh`, `_lib_verify.sh`. Keep `CLAUDE.md` (refresh contents).

**Acceptance:** `sproot new test-sprite` against this repo produces a sprite functionally identical to what `setup.sh` produces today. Any intentional differences listed in the migration notes.

---

### Phase 7: Release pipeline

Cut versioned binaries.

**Deliverables:**
- `.goreleaser.yaml`:
  - Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
  - Archives: tar.gz for unix, zip for windows
  - Checksums + sigstore signing
- `.github/workflows/release.yml`: triggers on `v*` tags
- `install.sh`: one-line installer for unix hosts. Detects OS/arch, downloads the right tarball, drops the binary in `/usr/local/bin` (root) or `~/.local/bin` (user)
- `README.md` on the sproot repo: install instructions, quickstart

**Acceptance:** `git tag v0.1.0 && git push --tags` produces a GitHub release with all binaries, checksums, and signatures. `curl -fsSL https://raw.githubusercontent.com/justanotherspy/sproot/main/install.sh | sh` installs a working binary on Linux and macOS.

---

### Phase 8: Docs

Final user-facing docs across both repos.

**Deliverables:**
- `sproot/README.md`: install, quickstart (`sproot config init`, set values, `sproot new my-sprite`)
- `sproot/docs/modules.md`: every module type with YAML example and notes
- `sproot/docs/config.md`: full `sproot.yaml` reference
- `sproot/docs/host-config.md`: `~/.sproot/config` reference
- `sproot/docs/auth-model.md`: the host-key + sprite-app-token split, why, security tradeoffs
- `sprite/README.md`: "this is a sproot config repo. fork it or write your own."

**Acceptance:** all internal doc links resolve. A new user can go from zero to a working sprite by following only the README.

---

## Cross-cutting constraints

- **No emdashes anywhere** in code comments, error messages, logs, or docs. Use parens or commas.
- **Single binary.** `sproot` routes by subcommand. No separate host/sprite binaries.
- **Embedding strategy.** Files referenced in `sproot.yaml` live in the config repo, not embedded in the sproot binary. The binary embeds only its own help text, version, and default schemas.
- **Host sprite interaction uses sprites-go SDK.** Never shell out to the `sprite` CLI from Go code. Use `github.com/superfly/sprites-go` for all sprite lifecycle and exec operations.
- **Bracket phases (pre/post sprite-env checkpoints) are a CLI concern, not a module type.** They wrap the whole setup run and never abort it.
- **The current `pre.sh` is reference only.** Don't port it; the built-in verify phase covers the same ground.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
