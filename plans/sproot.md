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

## Completed phases (0-16)

All phases through 16 are done and merged.

### Foundation (phases 0-7)

| Phase | What was built |
|-------|---------------|
| 0 | Scaffold: go.mod (Go 1.25), cobra, Makefile, CI (build-and-test + lint) |
| 1 | Config schema: `SprootConfig`, `HostConfig`, all phase config structs, custom YAML unmarshaling |
| 2 | Phase engine: `Phase` interface, runner (`--only`, `--force`), state file, registry |
| 3 | 17 module types: apt, uv_tool, go_install, cargo_install, binary_release, corepack, rust_components, docker, sprite_service, git_identity, ssh_setup, gh_token, file_template, rc_block, repo_clone, claude_settings, cmd |
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

### Phase 18: Intelligence and completion

#### 18a. llm.txt and agent-context.md after setup

After `sproot setup` completes inside a sprite, write a summary of what ran to `/.sprite/llm.txt` and `/.sprite/docs/agent-context.md`. Gives Claude Code instant context about the environment.

**Implementation:** New `internal/sprite/llmtxt.go` exports `writeLLMContext(l *log.Logger, state *phase.State) error`. Iterates `state.Phases`, collects `DidWork=true` records, writes a timestamped block using a module-description table keyed by phase `Type`. Hook after `runner.Run(ctx)` in `internal/sprite/setup.go` (non-fatal if write fails). Unit test in `internal/sprite/llmtxt_test.go`; integration test step verifies non-empty output via `sproot exec <name> cat /.sprite/llm.txt`.

#### 18b. Token scope documentation

Add a "Required GitHub token scopes" callout to `docs/modules.md` under `gh_token` and `ssh_setup` sections. Add a brief note to `README.md` prerequisites.

- `gh_token`: no minimum scope required by sproot; match whatever the user wants `gh` to do (typically `repo`, `read:org`)
- `ssh_setup`: requires `admin:public_key` and `admin:ssh_signing_key` on a classic PAT

#### 18c. Config init org auto-select

After the user provides `token_env` in interactive `config init`, check if the Sprites SDK exposes an org-listing method. If it does, call the API, print results, and prompt for selection; set `default_org` on selection; fall back to blank prompt on API failure or no results. Drop this item if no method exists.

**Files:** `internal/host/config.go`, `internal/host/config_test.go`.

#### 18d. Release workflow test

1. `goreleaser release --clean --snapshot` to confirm all 5 platform archives build
2. Push a test tag (e.g. `v0.0.0-rc1`) to trigger `release.yml` with cosign signing
3. Verify: checksums.txt, all archives, cosign bundle present in release assets
4. Delete the test tag and release
5. Fix `.goreleaser.yaml` or `release.yml` if anything fails

#### Deferred intelligence items (Phase 14)

- **14b.** Claude skill for sproot usage: generate `sproot.yaml` from description, validate/explain configs, suggest modules.
- **14c.** Script-to-`sproot.yaml` converter skill: map apt-get install -> apt, pip install -> uv_tool, curl-pipe-sh -> binary_release or cmd, git config -> git_identity, etc.

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

## Suggested order of execution

1. **Phase 17** (code quality and bug fixes, NEXT)
   - PR A first: push env forwarding (CRITICAL)
   - PR B: HTTP timeouts
   - PR C: quick wins bundle
2. **Phase 18** (intelligence and completion; 18a + 18c independent of 18b + 18d)
3. **Phase 14 deferred** (Claude skills; after everything else is stable)

After each PR: `make check` and `./sproot validate --path internal/config/testdata/sproot.yaml`.

---

## Cross-cutting constraints

- **No emdashes anywhere** in code comments, error messages, logs, or docs. Use parens or commas.
- **Single binary.** `sproot` routes by subcommand. No separate host/sprite binaries.
- **Embedding strategy.** Files referenced in `sproot.yaml` live in the config repo, not embedded in the sproot binary. The binary embeds only its own help text, version, and default schemas.
- **Host-sprite interaction uses sprites-go SDK.** Never shell out to the `sprite` CLI from Go code.
- **Idempotency is per-phase, not driven by the state file.** State file is for `--status` and forensics.
- **Feature branches for PRs.** Each body of work goes on a feature branch and merges via PR. Never push directly to main.
