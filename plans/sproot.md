# sproot: plan

A Go CLI for bootstrapping sprite.dev sprites from a user-owned config repo.
Replaces the current bash-based `sprite` repo with a generic, reusable tool.

## Two-repo model

- **`justanotherspy/sproot`** (new): the CLI source + release binaries. Generic, no personal config.
- **`justanotherspy/sprite`** (existing): becomes a reference *config repo*. Holds a `sproot.yaml` plus accompanying files (statusline.py, claude settings, etc). Anyone can fork it or write their own.

## Host file layout

```
~/.sproot/
└── config.yaml      # YAML: sproot_config_source, sproot_config_repo, sproot_config_ref,
                     #       sproot_config_path, sproot_config_local_path, token_env,
                     #       gh_token_env, default_org
```

Canonical field reference: `internal/config/schema.go` `HostConfig` struct.

## End-to-end flow

```
sproot new my-sprite
  ├─ reads ~/.sproot/config.yaml
  ├─ sprites.New(token).CreateSprite(ctx, "my-sprite", nil)
  ├─ sprite.Filesystem().WriteFile("/usr/local/bin/sproot", binaryBytes, 0755)
  ├─ sprite.Command("sproot", "setup", "--config-repo", url).Run()   // streams output
  └─ in-sprite: clone config repo, read sproot.yaml, run phases, verify
```

---

## Completed phases (0-20)

All phases through 20 are done and merged. Phases 0-16 (foundation and hardening) are
summarized in the two tables below; phases 17-20 (and the late-shipping Phase 14 skills work)
follow as their own sections.

### Foundation (phases 0-7)

| Phase | What was built |
|-------|---------------|
| 0 | Scaffold: go.mod (Go 1.25), cobra, Makefile, CI (build-and-test + lint) |
| 1 | Config schema: `SprootConfig`, `HostConfig`, all phase config structs, custom YAML unmarshaling |
| 2 | Phase engine: `Phase` interface, runner (`--only`, `--force`), state file, registry |
| 3 | 18 module types: apt, uv_tool, go_install, cargo_install, binary_release, corepack, rust_components, docker, sprite_service, git_identity, ssh_setup, gh_token, file_template, rc_block, repo_clone, claude, npm, cmd (claude replaced claude_settings in Phase 19; npm added in Phase 17) |
| 4 | `sproot setup` in-sprite: clone config repo, load sproot.yaml, run phases, verify |
| 5 | Host CLI: new, destroy, status, config, validate; sprites-go SDK client interfaces |
| 6 | justanotherspy/sprite converted to a config repo (separate repo; ongoing) |
| 7 | Release pipeline: goreleaser, sigstore signing, install.sh |

### Hardening and features (phases 8-16)

| Phase | Key changes | PR |
|-------|-------------|-----|
| 8 | Doc accuracy; env block forwarding in `sproot new`; `template: true` opt-in; cmd `name` field; binary checksum fields; `validate` also checks host config | #17 |
| 9 | rc_block dual-shell idempotency; binary_release GitHub API auth + HTTP timeouts; ssh_setup idempotency gap; cloneOrPull handles changed remote URL | #18 |
| 10 | Cross-arch injection: downloads Linux/amd64 release tarball at runtime on non-Linux hosts (`internal/host/fetch.go`) | #21 |
| 11 | Interactive config init; console command; list command; auto-setup prompt; `--debug` flag; pre-flight sproot.yaml validation | #22 |
| 12 | exec, upgrade, checkpoint/checkpoints/restore; `checkpoint_after_setup`; `status --host`. Note: `Upgrade` on `SpriteHandle` and `UpgradeSprite` on `SpritesClient` were added here but later removed in Phase 15 when `sproot upgrade` switched to running `sprite upgrade` inside the sprite. | #23 |
| 13 | Multi-target (targets/extends with inheritance + cycle detection); local path config source (`config_source: local`); `sproot push` with parallel execution and pre-push checkpointing | #28 |
| 15 | Remove unsupported create flags (--ram-mb, --cpus, --region, --storage-gb); `upgrade` runs `sprite upgrade` inside sprite; exec `--env`; HostConfig fields renamed to `sproot_config_*` prefix; `--skip-console` renamed from `--no-console` | various |
| 16 | Multi-phase + multi-target CI jobs; `--skip-verify`; `validate --strict`; config init source-first UX; smart verify (skips checks for tools whose phase wasn't in `--only`); `state.OnlyFilter`; `make e2e` | various |

### Key decisions from completed phases

- `sproot.yaml` uses flat phase format (not nested sub-keys as earlier docs showed)
- Flat `phases:` list is treated as implicit `default` target (backward compat; Q7 resolved)
- Idempotency is per-phase; state file is for `--status` and forensics only
- Host config stores env var **names**, not token values; tokens stay in the shell environment
- Single binary; `sproot` routes by subcommand; no separate host/sprite binaries
- `file_template` rendering is opt-in via `template: true` (Q2 resolved)
- Each sprite generates its own SSH keypair; `sproot destroy` removes it from GitHub
- `upgrade` runs `sprite upgrade` inside the sprite (SDK VM upgrade method not supported)
- Binary injection for non-Linux hosts downloads Linux/amd64 release tarball at runtime (Q6 resolved)
- `sproot push` always uses `--force`; checkpoints before pushing by default (`--no-checkpoint` to skip)

---

### Phase 17: Code quality and bug fixes (findings round 2) — DONE

Second-pass review findings. Three PRs recommended; ship in order A, B, C.

#### PR A: CRITICAL — push env forwarding (own PR, ship first)

**17a.** `internal/host/push.go` `pushOne` passes `nil` for env to `RunCommand`. The `env:` block from `sproot.yaml` and `gh_token_env` from `~/.sproot/config.yaml` are both silently dropped. `ssh_setup` and `gh_token` phases run without `GH_TOKEN` and degrade to warnings instead of errors.

**Fix:**
1. Extract `buildSpriteEnv(ghToken string, envBlock []string) []string` helper from `new.go`.
2. In `pushOne`, resolve `ghToken` from `cfg.GHTokenEnv` and call `buildSpriteEnv` before `RunCommand`.
3. Add `TestRunPush_ForwardsEnvBlock` and `TestRunPush_ForwardsGHToken` mirroring `new_test.go`.

Cross-reference: `TestRunNew_EnvBlockForwarded` and `TestRunNew_InjectsBinaryAndForwardsGHToken` in `internal/host/new_test.go` show the expected test pattern.

#### PR B: HTTP timeouts (own PR)

**17b.** `internal/host/destroy.go` `deleteGHKey` and `internal/phase/modules/ssh_setup.go` `postGHKey` both use `http.DefaultClient` with no timeout. If GitHub's API hangs, these commands hang indefinitely.

**Fix:** Add `var ghAPIClient = &http.Client{Timeout: 30 * time.Second}` to each file (consistent with `tagClient` in `binary_release.go`) and replace `http.DefaultClient.Do(req)` with `ghAPIClient.Do(req)` in both.

#### PR C: Quick wins bundle

**17c.** Consolidate duplicate `"sproot"` constant: remove `sprootLabel` from `internal/host/new.go`; update new.go, list.go, push.go, outdated.go to use `labelBase` from `labels.go`.

**17d.** `ConfigSHA` in `internal/host/labels.go` formats full 64-char hex then slices. Replace with `fmt.Sprintf("%x", h[:6])` for the same 12-char result.

**17e.** `Labels()` in `internal/host/labels.go` always emits `sproot-target=` even when `Target` is empty. Move `labelTarget` to the conditional block (matching `Repo` and `Ref` handling); update `labels_test.go`.

**17f.** README commands table for `sproot push` is missing `--only <type>`. Add it to the push row description.

**17g.** Expand MIGRATION.md module gaps into concrete improvement items (cross-reference `MIGRATION.md` "Known cmd workarounds"):
- **17g1** `apt`: add `symlinks` field for post-install symlinks (e.g. bat/fd)
- **17g2** `uv_tool`: auto-install uv when absent (currently fails if uv not present)
- **17g3** `uv_tool`: add `pkg` field for cases where package name differs from binary name (e.g. garlic)
- **17g4** `binary_release`: add `{x64_arch}` and `{x86_64_arch}` template variables (gitleaks uses `x64`, hadolint uses `x86_64`)
- **17g5** `docker`: add `daemon_json` config field for configuring the Docker daemon
- **17g6** `repo_clone`: add support for full git URLs with explicit `dest`. Currently only accepts `owner/repo` shorthand (clones via SSH to `<base_dir>/<repo>`). New long form: each entry may be either the existing string (`owner/repo`) or a struct `{url: "https://...", dest: "~/my-dir"}`. When `url` is used, `dest` is optional and defaults to `~/<repo-name>` (last path component of URL, minus `.git`). Custom YAML unmarshaling on `RepoCloneEntry` handles both forms. Files: `internal/config/schema.go` (new `RepoCloneEntry` union type + unmarshaling), `internal/phase/modules/repo_clone.go` (clone URL + dest resolution), `internal/phase/modules/repo_clone_test.go`, `docs/modules.md`.
- **17g7** new `npm` module: runs `npm install` in a target directory. Config fields: `dir` (required, path expanded with `~`). Idempotency: checks `test -d <dir>/node_modules`. Files: `internal/config/schema.go` (new `NpmConfig` struct and `PhaseConfig.Npm` field), `internal/phase/modules/npm.go`, `internal/phase/modules/npm_test.go`, `docs/modules.md`, integration test entry.
- **17g8** `sprite_service`: add `http_port` (optional int) and `needs` (optional `[]string`) fields to the service registration body. Both are omitted from the JSON when zero/nil (use `omitempty`). Files: `internal/config/schema.go` (update `SpriteServiceConfig`), `internal/phase/modules/sprite_service.go` (update body marshal), `internal/phase/modules/sprite_service_test.go`, `docs/modules.md`.

**Target sproot.yaml after 17g6-8** (openclaw-builder, no `cmd` modules):
```yaml
phases:
  - type: apt
    packages:
      - nodejs
      - npm

  - type: repo_clone
    repos:
      - url: https://github.com/theoctopusperson/openclaw-sprite-builder.git
        dest: ~/openclaw-builder

  - type: npm
    dir: ~/openclaw-builder

  - type: file_template
    src: files/openclaw-start.sh
    dest: ~/openclaw-builder/start.sh
    mode: "0755"

  - type: sprite_service
    service: openclaw-builder
    cmd: ~/openclaw-builder/start.sh
    http_port: 8080
```

The `files/openclaw-start.sh` in the config repo contains:
```bash
#!/bin/bash
cd ~/openclaw-builder
exec node server.js
```

**17h.** Housekeeping: mark `plans/findings.md` as superseded (add `> SUPERSEDED` header); remove stale Phase 13 done-items from `plans/todo.md`.

#### Deferred (OPEN QUESTIONS)

**17i.** `currentConfigSHA` in `internal/host/push.go` does `git clone --depth 1` into a temp dir on every `sproot outdated` call. Add a TODO comment in the function. Options if it becomes a complaint: cache SHA + timestamp in `~/.cache/sproot/sha-cache.json`, or use `git ls-remote` for a cheap commit SHA check before cloning.

**17j.** For git config sources, `sproot new` clones twice: host-side in `readEnvBlock` (to resolve env vars) and in-sprite in `RunSetup`. The in-sprite clone is unavoidable. The host clone is hard to skip without `git archive` support (GitHub HTTPS does not support `git archive --remote`). Defer unless startup time becomes a complaint.

---

### Phase 17g6-8 module additions — DONE

All three shipped and tested: `repo_clone` accepts full git URLs with explicit `dest` (`RepoCloneEntry` union type in `internal/config/schema.go`); new `npm` module (`internal/phase/modules/npm.go`); `sprite_service` gained optional `http_port` and `needs` fields. 18 module types are now registered.

### Phase 18: Intelligence and completion — 18a + 18b DONE; 18c dropped; 18d deferred

#### 18a. Post-setup summary inside the sprite — DONE

After `sproot setup` completes inside a sprite, it writes a summary of what ran to `/.sprite/docs/sproot-setup.md`, giving Claude Code instant context about what sproot installed.

**Non-destructive by design.** The sprite platform ships its own `/.sprite/llm.txt` (environment overview, security rules, services/checkpoints API) and `/.sprite/docs/agent-context.md` (full reference incl. network policy). sproot must NOT overwrite these. Instead it writes its own `docs/sproot-setup.md` and appends an idempotent managed pointer block (delimited by `<!-- BEGIN SPROOT --> / <!-- END SPROOT -->`) to the platform `llm.txt` so agents reading the canonical entrypoint discover the summary.

**Implementation:** `internal/sprite/llmtxt.go` exports `renderLLMContext(state)` (pure) and `writeLLMContext(l, state, baseDir)`. It collects `DidWork=true` records and renders a timestamped markdown block from a module-description table keyed by phase `Type`, then writes `docs/sproot-setup.md` and updates the `llm.txt` pointer block (`appendPointerBlock`, which strips any prior block first for idempotency). `Runner.LastState()` (added in `internal/phase/runner.go`) exposes the final state; `internal/sprite/setup.go` calls `writeLLMContext` after `runner.Run(ctx)` (non-fatal if the write fails; setup runs as root so it can write under `/.sprite`). Unit tests in `internal/sprite/llmtxt_test.go` (including a platform-content-preservation/idempotency test); an integration step in the `multi-phase` job verifies `/.sprite/docs/sproot-setup.md` exists and that the platform `llm.txt`/`agent-context.md` are preserved with the sproot pointer added.

#### 18b. Token scope documentation — DONE

`docs/modules.md` documents scopes under `gh_token` and `ssh_setup`; `README.md` gained a consolidated "GitHub token scopes" subsection under Host config.

- `gh_token`: no minimum scope required by sproot; match whatever the user wants `gh` to do (typically `repo`, `read:org`)
- `ssh_setup`: requires `admin:public_key` and `admin:ssh_signing_key` on a classic PAT

#### 18c. Config init org auto-select — DROPPED

The sprites-go SDK exposes no org-listing method (only `CreateSpriteWithOrg` and an `OrgInfo` returned alongside sprite listings), so per the original "drop if no method exists" guidance this item is dropped. `default_org` remains a free-text prompt in `config init`.

#### 18d. Release workflow test — DEFERRED

1. `goreleaser release --clean --snapshot` to confirm all 5 platform archives build
2. Push a test tag (e.g. `v0.0.0-rc1`) to trigger `release.yml` with cosign signing
3. Verify: checksums.txt, all archives, cosign bundle present in release assets
4. Delete the test tag and release
5. Fix `.goreleaser.yaml` or `release.yml` if anything fails

---

### Phase 14: Intelligence (skills) — DONE

(Numbered 14 but shipped late, after Phase 19, as PR #47.)

Shipped as a Claude Code plugin marketplace inside this repo (`.claude-plugin/marketplace.json`)
with one plugin (`plugins/sproot/`) holding two skills, rather than baking conversion into the
binary: bash is too irregular for a deterministic parser, so the skills give Claude a playbook +
an authoritative module map and let it do the semantic translation, then validate the result with
the real `sproot validate`.

- **14c. `script-convert`**: turns a setup bash script into a `sproot.yaml` plus companion files.
  Coalesces (one `apt`, one `repo_clone`, one `rc_block`), extracts heredoc/`cat >`/`tee` bodies
  into a `files/` layout (`file_template`), folds shell-rc appends into one `rc_block` companion,
  forwards secrets via the `env` block (never inline), drops `git config user.name/email`
  (covered by `identity`), and falls back to `cmd` (with a `check:`) only when nothing structured
  fits. Decision table lives in `plugins/sproot/reference/module-map.md`.
- **14b. `author-config`**: generate from a description, explain phase-by-phase, interpret/fix
  `sproot validate` errors, suggest modules. Shares the reference docs.
- **Shared reference**: `reference/module-map.md` (idiom -> module) and `reference/module-schema.md`
  (exact flat field shapes) mirror `docs/modules.md` + `internal/config/schema.go`; the
  CLAUDE.md workflow now requires keeping them in sync when modules change.
- **Verification**: seven golden fixtures under `skills/script-convert/examples/` (input.sh +
  expected/), each passing `sproot validate`. CI (`plugins.yml`) runs `claude plugin validate
  --strict` on the marketplace and plugin, and `sproot validate` on every fixture.

---

### Phase 19: Module gaps to drain `cmd` blocks — DONE

Motivated by auditing the (out-of-date) `justanotherspy/sprite` `sproot.yaml`, whose `cmd`
blocks worked around missing module features. A fresh-sprite recon confirmed: no default
`/etc/docker/daemon.json` (dir absent), the platform recommends `storage-driver: overlay2`
and runs dockerd as a sprite service (no systemd), `uv tool` binaries land in `~/.local/bin`
which is already on PATH (so uv_tool needs no fix), and apt installs `bat`/`fd` as
`batcat`/`fdfind`. Captured the platform `/.sprite/llm.txt`, `llm-dev.txt`, and
`docs/{agent-context,docker,services}.md` into `sprites-artefacts/` for reference.

- **binary_release `version`**: optional template for the `{version}` token (default raw tag);
  new `{tag}`/`{tag_no_v}` vars. The download URL path always uses the raw tag. Drains the
  gitleaks `cmd` (and fixes cosign, whose `.deb` also drops the leading `v`).
- **binary_release `arch_map` + `{arch_alias}`**: covers any arch naming the fixed vars miss
  (hadolint: `x86_64` on amd64, `arm64` on arm64). Static validation requires `arch_map` when
  `{arch_alias}` is used; the per-GOARCH key is a runtime check (host arch may differ from sprite).
- **binary_release `cosign`**: keyless `cosign verify-blob` of a signed checksums file, then
  verifies the asset against it. Drains the trufflehog `cmd`. Hard-errors if cosign is absent;
  the cosign-installing phase must precede phases using a `cosign:` block (runner is list-ordered).
- **`claude` module replaces `claude_settings`**: settings deep-merge + `upgrade` + a managed
  CLAUDE.md block (HTML-comment sentinels) sourced from a config-repo file (optional Go template).
  Drains the `claude upgrade` `cmd` and folds in settings. Shared `deepMerge`/`jsonStr` moved to
  `jsonmerge.go`; `rc_block` block helpers generalized to take begin/end sentinels.
- **docker daemon.json**: deep-merges into any existing file (absent file -> empty start; corrupt ->
  error); sprite-env-aware (no `systemctl restart` on a sprite; `sprite_service` starts dockerd with
  the merged config). Drains the daemon.json `cmd`.
- **apt symlinks**: `~` expansion + parent `mkdir`, so the bat/fd shims move off `cmd`.
- **privilege fix (`runPrivileged`)**: `sproot setup` runs as the unprivileged `sprite` user, so
  `apt-get install` and `dpkg -i` failed (superuser required) — the apt phase only "worked" in CI
  because the test packages were pre-installed. Added a shared `runPrivileged` that prepends `sudo -n`
  when not root (sudo is passwordless on sprites); applied to `apt-get install` and the `dpkg` install.
  Verified end-to-end on a real sprite: cosign (dpkg), gitleaks (version), hadolint (arch_map),
  trufflehog (cosign verify-blob "Verified OK"), claude (settings deep-merge preserved the platform's
  hooks/permissions; CLAUDE.md template-rendered block), and docker (daemon.json merged with
  storage-driver overlay2, systemctl skipped under sprite-env, dockerd registered via sprite_service).

Follow-up (user-owned, separate repo): rewrite `justanotherspy/sprite/sproot.yaml` to use these
features (examples in `docs/modules.md` and `MIGRATION.md`). Remaining legitimate `cmd` blocks:
flyctl install, `garlic setup --defaults`, and shell-completion generation.

### Phase 20: Self-update and daily update notifier — DONE

The CLI now keeps itself current. All logic lives in `internal/host/selfupdate.go` (host-side,
no sprite interaction), wired through a new `sproot self-update` command and the root command's
`PersistentPostRunE`.

- **Daily cached check**: a JSON cache at `~/.sproot/update-check.json`
  (`{checked_at, latest_version}`) records the last successful query of GitHub's
  `releases/latest`. `cachedLatestVersion` serves the cached value while it is younger than 24h
  and otherwise refetches and rewrites the cache (atomic temp+rename). The query carries a
  `User-Agent` (the GitHub API 403s without one) and uses a 3s-timeout client so the once-a-day
  refresh can never noticeably stall a command.
- **Notifier**: `NotifyUpdateAvailable` runs from `PersistentPostRunE` after every command
  except `setup` (runs in-sprite) and `self-update` (does its own check). It prints a two-line
  notice to stderr only (never stdout, so pipelines are unaffected). It is fully best-effort:
  any error (network, parse, cache I/O) is swallowed. `dev`/unparseable versions are silent, and
  `SPROOT_NO_UPDATE_CHECK` disables it entirely (used by the CI smoke step).
- **`sproot self-update`**: always re-checks upstream (ignoring the cache), and on a real upgrade
  downloads `sproot_<ver>_<os>_<arch>.{tar.gz,zip}` for the host's `runtime.GOOS/GOARCH`, verifies
  it against the published `_checksums.txt`, extracts the binary (reusing `extractSprootFromTarGz`;
  a new `extractSprootFromZip` for Windows), and atomically swaps the running executable
  (`os.Rename` over the live inode on Unix; move-aside on Windows). It then clears the cache so the
  next command reflects the new version. `--check` reports availability without downloading.
  Dev builds refuse to self-update (no matching release); a 403 with `X-RateLimit-Remaining: 0`
  surfaces a friendly "rate limit exceeded" message.
- **Version comparison**: promoted `github.com/Masterminds/semver/v3` from indirect to direct;
  `isNewer` returns false on any unparseable version so the user is never nagged on bad data.
- **Testing**: 13 unit tests in `internal/host/selfupdate_test.go` cover version comparison,
  checksum verify (ok/mismatch/missing), tar.gz+zip extraction, cache round-trip/freshness/stale
  fallback, the notice matrix, and `RunSelfUpdate` (dev refusal, up-to-date, check-only,
  binary replacement + cache clear, download-error propagation) via `latestFn`/`fetchFn`/`execPath`
  seams. The live API path is not exercised in `integration.yml`: self-update touches no sprite,
  and the unauthenticated GitHub API is rate-limited per-IP on shared runners. Instead `ci.yml`
  has a network-free smoke step asserting the command is wired and refuses a dev build.

---

### Phase 21: Config SHA cache + shell_completion module — DONE

Two independent improvements drained from the open-items list.

#### Config SHA cache (17i + 17j host-side clone)

`sproot new`, `push`, and `outdated` previously did a full `git clone --depth 1` of the
config repo host-side on every invocation just to read `sproot.yaml` (resolve the env block and
compute the config SHA). Now a cache at `~/.sproot/config-cache.json` keyed by
`(repo, ref, configPath)` stores the last-cloned `sproot.yaml` content alongside the git commit
SHA the ref pointed at. `loadConfigBytes` (in `internal/host/configcache.go`) first runs a cheap
`git ls-remote <repo> <ref>`; on a cache hit (the ref still points at the cached commit) it returns
the cached content and skips the clone entirely. A cache miss, or any `ls-remote` failure, falls
back to the full clone and refreshes the cache, so behavior is never worse than before.

- Only the public `sproot.yaml` content is cached. The env block (which resolves to secret values)
  is re-resolved from the host environment on every call via `parseAndResolveEnv`; secrets never
  touch disk.
- `parseLsRemote` prefers `refs/heads/<ref>` over a same-named tag (mirroring `git clone --branch`)
  and the dereferenced `^{}` commit for annotated tags (so it matches `git rev-parse HEAD`).
- `readEnvBlock` is now a thin wrapper over `loadConfigBytes` + `parseAndResolveEnv`;
  `currentConfigSHA` benefits automatically. The cache write is atomic (temp+rename), mirroring the
  Phase 20 update-check cache.
- Tests: `internal/host/configcache_test.go` covers ls-remote ref selection, cache round-trip, a
  cache-hit-skips-clone path (overwrites the cached body with a sentinel and asserts it is returned
  without a re-clone), and cache invalidation when the ref advances (real local git repos).

This is the resolution of deferred items 17i and 17j (the in-sprite clone in `sproot new` is still
unavoidable; only the host-side clone is cached).

#### shell_completion module

New module (the 19th) that generates and installs shell completion scripts and wires the shell to
load them, draining a common `cmd` recipe (the `justanotherspy/sprite` config's completion block).

```yaml
- type: shell_completion
  completions:
    - command: sproot
      shells: [bash, zsh, fish]
    - command: gh
      shells: [bash, zsh]
      gen: "{command} completion {shell}"   # optional; this is the default
```

- Generation defaults to the cobra convention `{command} completion {shell}` (works for sproot, gh,
  kubectl, ...), with an optional per-entry `gen` template (tokens `{command}`/`{shell}`, split on
  whitespace and exec'd directly, no shell) for tools that emit completions differently.
- Installs to per-user dirs (no root): bash `~/.local/share/bash-completion/completions/<cmd>`,
  zsh `~/.zfunc/_<cmd>`, fish `~/.config/fish/completions/<cmd>.fish`. bash/fish auto-load; for zsh
  the module appends a managed `fpath`+`compinit` block to `~/.zshrc` (sentinels
  `# BEGIN/END SPROOT COMPLETIONS`, replaced not duplicated), reusing rc_block's `applyManagedBlock`.
- Idempotency: skips when all target files exist and (if zsh) the rc block is current.
- Files: `internal/config/schema.go` (`ShellCompletionConfig`/`ShellCompletionEntry`), `validate.go`,
  `internal/phase/modules/shell_completion.go` + test, dry-run entry in `integration_test.go`,
  `docs/modules.md`, plugin `reference/module-map.md` + `module-schema.md`, a `shell-completion`
  matrix job in `integration.yml` (generates sproot's own completions on a sprite), and a phase in
  `testdata/integration/sproot_tooling.yaml`.

---

### Examples and design notes (post-Phase 20)

These shipped after Phase 20 as docs/examples, not as new code phases.

- **OpenClaw builder example (#51)**: `examples/openclaw/` is a complete `sproot.yaml`
  (plus `files/start.sh`) that translates the upstream `openclaw-sprite-builder` `setup.sh`
  into modules. It exercises exactly the 17g6-8 additions end to end with no `cmd` blocks:
  `repo_clone` with a full git URL + explicit `dest`, `npm` install, `file_template` for the
  start script, and `sprite_service` with `http_port`. It is the worked example the Phase 19
  follow-up called for (a real config repo using the drained features), kept in-repo rather
  than in the separate `justanotherspy/sprite` repo. README links to it.

- **Agent on a sprite (#50)**: `plans/claude-agent.md` is a standalone design for bootstrapping
  a sprite that runs a Claude Code agent against a target repo from a single `sproot new`. Key
  finding: v1 needs no new module. The `env` block forwards `ANTHROPIC_API_KEY` /
  `CLAUDE_CODE_OAUTH_TOKEN` (non-interactive auth is just an env var, there is no `/login` to
  script), and `claude` + `gh_token` + `repo_clone` + a final `cmd` running `claude -p` cover
  the flow. A dedicated `claude_agent` module is proposed (workdir, prompt source, permission
  mode, blocking vs `sprite_service` background) but deferred per its own Q3 until the pattern
  recurs. If built, it becomes the candidate Phase 21 (see that doc for the file-touch list and
  open questions Q1-Q5).

---

## Open questions

All Q1-Q7 resolved.

| # | Question | Resolution |
|---|----------|------------|
| Q1 | env block forwarding | Implemented (option 1): resolve vars host-side, forward to `sproot setup` |
| Q2 | file_template opt-in | `template: true` field added (option 2) |
| Q3 | cmd name field | `name: string` field added to `CmdConfig` |
| Q4 | binary_release checksums | Both `checksum` and `checksum_asset` fields added |
| Q5 | validate commands | `sproot validate` also checks host config; `config validate` retained separately |
| Q6 | cross-arch binary injection | Download at runtime (option 2) |
| Q7 | multi-target backward compat | Flat `phases:` = implicit default target; no migration needed |

---

## Status and remaining work

All numbered phases (0-21) plus the Phase 14 skills work are done and merged. Nothing in the
core roadmap is outstanding. The only open item is a design that has not been committed to code:

- **17i / 17j** (DONE in Phase 21): the host-side config clone is now cached via `git ls-remote`
  (`~/.sproot/config-cache.json`). The in-sprite clone in `sproot new` remains (unavoidable).
- **Config-repo SSH bootstrap** (DONE): the in-sprite clone always rewrites a GitHub SSH
  config-repo URL (`git@github.com:...`) to HTTPS, because a freshly created sprite has no SSH key
  registered with GitHub and no `github.com` in `known_hosts` (the `ssh_setup` phase that would
  provision both lives *inside* the config repo, so it cannot gate the clone of that repo). Public
  repos clone anonymously; private repos use the forwarded `GH_TOKEN` via a transient `-c
  http.<url>.extraheader` arg (never persisted to `.git/config`). The host side keeps SSH when SSH
  works and only falls back to HTTPS when SSH is unreachable, so private-repo access via a real
  laptop's key is preserved. Both paths warn so the user can switch `sproot_config_repo` to HTTPS.
- **`validate` resolves the configured source** (DONE): `sproot validate` with no `--path` now
  resolves the sproot.yaml from the configured source (the git config repo via the cached
  `loadConfigBytes`, or the local config dir) instead of looking for `./sproot.yaml`, so it checks
  the same file `sproot new`/`push` use. `--path` still validates a specific local file, and a bare
  `validate` with no host config falls back to `./sproot.yaml`.
- **18d** (DONE): validated by the published `v0.1.0` release, which carries all 5 platform
  archives, `sproot_0.1.0_checksums.txt`, and the `..._checksums.txt.sigstore.json` cosign bundle.
  (Housekeeping: a stale untagged `v0.1.1` release-drafter *draft* exists and can be cleaned up.)
- **claude_agent module** (deferred): the agent-on-sprite design (`plans/claude-agent.md`).
  A candidate future phase if/when the `cmd` recipe recurs enough to justify a module.

After each PR: `make check` and `./sproot validate --path internal/config/testdata/sproot.yaml`.

---

## Cross-cutting constraints

- **No emdashes anywhere** in code comments, error messages, logs, or docs. Use parens or commas.
- **Single binary.** `sproot` routes by subcommand. No separate host/sprite binaries.
- **Embedding strategy.** Files referenced in `sproot.yaml` live in the config repo, not embedded in the sproot binary. The binary embeds only its own help text, version, and default schemas.
- **Host-sprite interaction uses sprites-go SDK.** Never shell out to the `sprite` CLI from Go code.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
- **Feature branches for PRs.** Each body of work goes on a feature branch and merges via PR. Never push directly to main.
