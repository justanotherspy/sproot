# Claude agent on a sprite: design

Goal: bootstrap a sprite that runs a Claude Code agent against a target repo with a
single host command. The operator runs `sproot new <name>`, and the sprite ends up with
Claude installed, authenticated non-interactively, a repo cloned, and an agent kicked off
against a supplied prompt.

This doc records how that is expressible today with existing modules, the tradeoffs, and a
proposed `claude_agent` module that packages the pattern.

## Key finding: no new module is required for a first version

The pieces already exist. The mechanic that makes "non-interactive login" trivial is the
`env` block plus the way setup inherits its environment:

- `sproot new` resolves the `env` block host-side and forwards the values to the in-sprite
  `sproot setup` process (`internal/host/new.go`, `buildSpriteEnv`).
- Every phase runs as a child process of `sproot setup`, so it inherits those env vars.
  This is exactly how `gh_token` reads `GH_TOKEN` (`internal/phase/modules/gh_token.go`).
- Claude Code reads `ANTHROPIC_API_KEY` (or a subscription token `CLAUDE_CODE_OAUTH_TOKEN`
  from `claude setup-token`) from the environment. So forwarding the token via `env` is the
  whole "login flow"; there is no interactive `/login` to script.

So a `cmd` phase that runs `claude -p "$PROMPT"` sees the auth token and the prompt directly.

## Single-command flow

```
sproot new garlic-agent --skip-console
  ├─ reads ~/.sproot/config.yaml
  ├─ clones config repo to resolve the env block, forwards ANTHROPIC_API_KEY / GH_TOKEN /
  │  SPROOT_AGENT_PROMPT from the host shell
  ├─ creates sprite, injects sproot binary, uploads config repo
  └─ runs `sproot setup` in-sprite (env forwarded):
       apt -> git_identity -> gh_token -> claude -> repo_clone -> cmd(run-agent)
                                                                   └─ claude -p "$PROMPT"
```

`--skip-console` is used so the host does not drop into a TTY; the agent runs as the final
setup phase and setup returns when it finishes.

## Module mapping

| Requirement | Module |
|-------------|--------|
| single command | `sproot new <name>` |
| non-interactive Claude auth | `env` block forwards `ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` |
| install / configure Claude | `claude` (settings, `upgrade`, managed `CLAUDE.md`) |
| GitHub auth (clone / push / PR) | `gh_token` |
| repo to work on | `repo_clone` |
| inject a prompt | `env` var (e.g. `SPROOT_AGENT_PROMPT`) read by a final `cmd` phase |
| run the agent | `cmd` running `claude -p`, or `sprite_service` for a background agent |

## Example sproot.yaml (existing modules only)

```yaml
schema_version: 1

identity:
  git_user_name: "Daniel Schwartz"
  git_user_email: "daniel.schwartz@hey.com"
  git_default_branch: main
  gh_username: justanotherspy

env:
  # Non-interactive Claude auth: an API key OR a subscription token
  # (from `claude setup-token`). Either is read from the environment.
  - from: ANTHROPIC_API_KEY
    as: ANTHROPIC_API_KEY
    required: true
  # GitHub auth so the agent can clone, push, and open PRs.
  - from: GITHUB_TOKEN
    as: GH_TOKEN
    required: true
  # The task, supplied fresh on each launch.
  - from: SPROOT_AGENT_PROMPT
    as: SPROOT_AGENT_PROMPT
    required: true

phases:
  - type: apt
    packages: [git, curl, jq, ripgrep]

  - type: git_identity
  - type: gh_token

  - type: claude
    upgrade: true
    claude_md: files/CLAUDE.md
    template: true

  - type: repo_clone
    base_dir: ~/work
    repos:
      - justanotherspy/garlic

  - type: cmd
    name: run-agent
    run: >
      cd ~/work/garlic &&
      claude -p "$SPROOT_AGENT_PROMPT"
      --permission-mode acceptEdits
      --output-format stream-json --verbose
      | tee ~/agent-run.log
```

Launch:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export GITHUB_TOKEN=ghp_...
export SPROOT_AGENT_PROMPT="Fix the failing tests in ./internal and open a PR."
sproot new garlic-agent --skip-console
```

## Tradeoffs

1. One-shot vs long-lived. A `cmd` phase runs the agent during setup and blocks until it
   finishes, which is clean for "do this task, open a PR, done." For a persistent or
   background agent, model it as a `sprite_service` so setup returns immediately and the
   agent runs as a managed service.

2. Auth persistence. The forwarded token is present during setup only. An interactive
   console opened afterward (`internal/host/new.go`, the `handle.Console` call) does not get
   the env, because `Console` is passed no env block. Headless one-shot runs need no
   persistence. Persisting for interactive use means writing the secret to disk
   (rc_block / file), a plaintext-secret tradeoff. On the hosted platform these vars are
   normally injected for you.

3. Permissions. An autonomous headless agent needs `--permission-mode acceptEdits` or
   `--dangerously-skip-permissions`. The sprite is a throwaway sandbox, so skipping
   permissions is defensible, but it is an explicit choice, not a default.

4. Prompt delivery. An env var (shown above) is best for "same setup, different task each
   run." Alternatives: a fixed prompt file baked into the config repo via `file_template`,
   or several canned agents expressed as `targets`.

## Proposed `claude_agent` module

A dedicated module is more ergonomic and CI-testable than a raw `cmd`. It is optional; the
`cmd` approach above works without it.

Proposed schema (`internal/config/schema.go`, `ClaudeAgentConfig`):

```yaml
- type: claude_agent
  workdir: ~/work/garlic        # required; cwd for the run
  prompt_env: SPROOT_AGENT_PROMPT  # env var holding the prompt (one of prompt_env/prompt/prompt_file)
  prompt: "..."                 # inline prompt (mutually exclusive with prompt_env/prompt_file)
  prompt_file: files/task.md    # config-repo-relative prompt file
  permission_mode: acceptEdits  # default | acceptEdits | plan | bypassPermissions
  skip_permissions: false       # maps to --dangerously-skip-permissions
  output: ~/agent-run.log       # optional; capture stream-json output here
  background: false             # false -> blocking cmd; true -> register as sprite_service
```

Behavior:
- Resolves the prompt from exactly one of `prompt`, `prompt_env`, `prompt_file`.
- Fails fast in `ShouldRun`/`Verify` if no Claude auth env var is present
  (`ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN`), with a clear error.
- `background: false` runs `claude -p` synchronously (like `cmd`).
- `background: true` registers a sprite-env service (reuse `sprite_service` plumbing).
- Idempotency: agent runs are side-effectful and not naturally idempotent. Default to
  "always run" (like `claude upgrade`), or gate on a sentinel file the agent writes when
  done. Document the choice; do not pretend a run is idempotent when it is not.

Files to touch when building it (per the repo workflow rules in CLAUDE.md):
- `internal/config/schema.go`: add `ClaudeAgent *ClaudeAgentConfig` to `PhaseConfig`, the
  struct, and the `UnmarshalYAML` switch.
- `internal/phase/modules/claude_agent.go`: the module, registered in `init`.
- `docs/modules.md`: module reference.
- `plugins/sproot/reference/module-map.md` and `reference/module-schema.md`: keep the
  script-convert / author-config skills in sync.
- Tests: unit test under `internal/phase/modules/`, a dry-run path in
  `internal/phase/modules/integration_test.go`, and an `integration.yml` job that exercises
  a real headless run against a sprite (a trivial prompt that touches a file, then assert
  the file exists). Cover both a passing run and the missing-auth error.

## Open questions

| # | Question | Notes |
|---|----------|-------|
| Q1 | API key vs subscription token | Support both; auth check accepts either env var. |
| Q2 | Idempotency model for agent runs | Always-run vs sentinel-file gate. Lean always-run, document it. |
| Q3 | Do we need `claude_agent` at all for v1? | No. Ship the `cmd` recipe first (example fixture), add the module if the pattern recurs. |
| Q4 | Persist auth for interactive sessions? | Out of scope for headless v1; revisit with a secret-on-disk decision. |
| Q5 | Integration test cost | A real headless run consumes model tokens in CI. Decide whether to gate it like the owner-triggered integration jobs. |
