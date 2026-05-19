# sproot: plan

A Go CLI for bootstrapping sprite.dev sprites from a user-owned config repo.
Replaces the current bash-based `sprite` repo with a generic, reusable tool.

## Two-repo model

- **`justanotherspy/sproot`** (new): the CLI source + release binaries. Generic, no personal config.
- **`justanotherspy/sprite`** (existing): becomes a reference *config repo*. Holds a `sproot.yaml` plus accompanying files (statusline.py, claude settings, etc). Anyone can fork it or write their own.

## Host file layout

```
~/.sproot/
├── config           # YAML: config_repo_url, private_key_path, defaults
└── private/
    └── id_ed25519   # user's sprite SSH key (loaded from password manager)
```

## End-to-end flow

```
sproot new my-sprite
  ├─ reads ~/.sproot/config
  ├─ sprite create my-sprite --skip-console
  ├─ sprite exec --file <key>:.ssh/id_ed25519 --file ./sproot:/usr/local/bin/sproot \
  │     sproot setup --config-repo <url>
  └─ in-sprite: clone config repo, read sproot.yaml, run phases, verify
```

## Phases (for execution)

Each phase is a discrete deliverable. Phases are roughly in dependency order. Tests are required inline per phase, not deferred.

---

### Phase 0: Scaffold `justanotherspy/sproot`

Create the new repo with Go module setup and CI.

**Deliverables:**
- `go.mod` (Go 1.23+), cobra for command routing
- Layout:
  ```
  cmd/sproot/main.go
  internal/config/        # yaml schema + loader
  internal/phase/         # phase interface + state + runner
  internal/phase/modules/ # one file per module type
  internal/host/          # host-side command implementations
  internal/sprite/        # in-sprite command implementations
  pkg/log/                # +/-/!/x logging conventions
  ```
- `.github/workflows/ci.yml`: build + test + golangci-lint on push
- Minimal `cmd/sproot/main.go` that wires up cobra and prints help

**Acceptance:** `go build ./...` and `go test ./...` succeed. CI passes on first push.

---

### Phase 1: Config schema

Define `sproot.yaml` (in the config repo) and `~/.sproot/config` (on the host).

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
config_repo: git@github.com:justanotherspy/sprite.git
config_ref: main
private_key: ~/.sproot/private/id_ed25519
default_org: ""
```

**Deliverables:**
- `internal/config/schema.go`: Go structs with `yaml:""` tags
- `internal/config/load.go`: loaders for both files
- `internal/config/validate.go`: required-field checks, type checks, unknown-module-type detection
- `testdata/sproot.yaml` + `testdata/host_config.yaml`
- Unit tests covering happy path + each validation failure

**Acceptance:** sample YAMLs parse, invalid ones fail with clear errors that name the offending field.

---

### Phase 2: Phase engine

The runtime that executes phases listed in `sproot.yaml`.

**Deliverables:**
- `internal/phase/phase.go`:
  ```go
  type Phase interface {
      Type() string
      Name() string                          // human label for logs
      ShouldRun(ctx *Context) (bool, error)  // idempotency check
      Run(ctx *Context) error
      Verify(ctx *Context) error             // post-run validation
  }
  ```
- `internal/phase/context.go`: shared state (config repo path, identity, logger, dry-run, force flags)
- `internal/phase/runner.go`: orchestrates the phase list, tracks did-work / skipped / failed buckets, supports `--only` and `--force`
- `internal/phase/state.go`: reads/writes `~/.config/sproot/state.json`
- `internal/phase/registry.go`: module type registration via `init()`
- `pkg/log/`: structured logger matching the current `+`/`-`/`!`/`x` visual conventions
- Tests for runner with dummy phases (one passes, one fails, one skips, one is forced)

**Acceptance:** dummy phases run end-to-end. State file gets written. `--only` runs a single phase. `--force` overrides idempotency.

---

### Phase 3: Phase module implementations

Every module type the current `setup.sh` needs. Each module lives in one file under `internal/phase/modules/`, registered via `init()`.

**Module list:**
- `apt` — install apt packages
- `uv_tool` — install via `uv tool install`
- `binary_release` — download from `github.com/<repo>/releases/latest`. Supports asset templating (`{version}`, `{arch}`, `{goos}`, `{dpkg_arch}`) and install methods (`dpkg`, `tar+install`, `raw`)
- `corepack` — enable + pre-activate pnpm and yarn
- `rust_components` — pin stable, install clippy/rustfmt/rust-analyzer
- `docker` — docker-ce install + `/etc/docker/daemon.json`
- `sprite_service` — register a `sprite-env` service (dockerd today, anything else later)
- `git_identity` — user.name, user.email, default branch, aliases, signing config
- `ssh_setup` — chmod the injected key, populate known_hosts, derive pubkey via `ssh-keygen -y -f`, write `allowed_signers`
- `gh_token` — export `GH_TOKEN` from the sprite-attached app token; verify `gh auth status`
- `file_template` — copy a file from the config repo to a dest path, with optional Go-template execution against the identity struct
- `rc_block` — write a sentinel-delimited block to `.bashrc` and `.zshrc` from a source file. Strips legacy blocks.
- `repo_clone` — clone a list of `git@github.com:owner/repo` into a base dir
- `claude_settings` — deep-merge a JSON object into `~/.claude/settings.json`
- `cmd` — escape hatch. Run an arbitrary command, with an optional idempotency-check command. For things that don't deserve a module.

**Deliverables:**
- One Go file per module, plus unit tests
- Each module documents its YAML schema in a doc comment at the top
- `docs/modules.md` reference

**Acceptance:** unit tests for each module pass. An integration test constructs a small YAML using each module type and runs it under `--dry-run`.

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

The commands the user runs on their laptop.

**Deliverables:**
- `sproot new <name>`:
  - Reads `~/.sproot/config`
  - Runs `sprite create <name> --skip-console`
  - Runs `sprite exec --file <key>:.ssh/id_ed25519 --file <binary>:/usr/local/bin/sproot -s <name> sproot setup --config-repo <url>`
  - Streams output, returns the sprite's exit code
- `sproot destroy <name>` — calls `sprite destroy <name>`. No GitHub-side cleanup needed under the one-key model.
- `sproot status <name>` — `sprite exec`s `sproot setup --status` remotely, prints the table
- `sproot config init` — writes a skeleton `~/.sproot/config`
- `sproot config validate` — validates `~/.sproot/config` and optionally fetches and validates the config repo's `sproot.yaml`
- Mode routing: single binary detects host vs in-sprite by command name (`new` vs `setup`). No separate binary.

**Acceptance:** `sproot new my-sprite` produces a working sprite end-to-end against `justanotherspy/sprite` as the config repo. `sprite console -s my-sprite` lands in a usable shell.

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
- **Bracket phases (pre/post sprite-env checkpoints) are a CLI concern, not a module type.** They wrap the whole setup run and never abort it.
- **The current `pre.sh` is reference only.** Don't port it; the built-in verify phase covers the same ground.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
