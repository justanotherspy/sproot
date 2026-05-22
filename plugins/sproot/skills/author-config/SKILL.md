---
name: author-config
description: Generate, explain, validate, or fix a sproot.yaml, and suggest the right sproot module for a need. Use when the user wants help writing a sproot config, asks what a sproot.yaml does, hits a sproot validate error, or asks which sproot module to use.
argument-hint: "[generate|explain|validate|suggest] [description-or-path]"
allowed-tools: Read, Write, Grep, Glob, Bash
---

# Author and explain sproot configs

You help users work with `sproot.yaml`: write one from a description, explain an existing one,
diagnose `sproot validate` errors, and recommend modules. For converting an existing bash setup
script, use the `script-convert` skill instead.

## Load the reference first

Read these bundled references (via `${CLAUDE_PLUGIN_ROOT}/reference/`, or relative at
`../../reference/`):

- `reference/module-schema.md`: exact fields for every module and the top-level shape.
- `reference/module-map.md`: what each module is for (the right-hand column).

Field names and the flat phase structure must match exactly. The canonical source in the repo is
`docs/modules.md` and `internal/config/schema.go`; consult them when a detail is missing here.

## Modes

Pick the mode from `$1` or infer it from the request.

### generate: config from a description
1. Clarify the target environment if ambiguous (languages, tools, services, repos to clone).
2. Emit `schema_version: 1`, an `identity:` block (placeholders the user fills in), an `env:`
   block if any secret-dependent module is used (`gh_token`, `ssh_setup`), and `phases:`.
3. Prefer structured modules over `cmd`. Order phases install-before-use.
4. Write to a path the user chooses (default `./sproot.yaml`), then validate (below).

### explain: narrate an existing config
1. Read the given `sproot.yaml`.
2. Walk the phases in order; for each, state the module, what it installs/configures, and any
   idempotency or privilege note from `module-schema.md`/`docs/modules.md`.
3. Call out `identity`, `env`, and target inheritance (`extends`) if present.

### validate: run and interpret
1. Build sproot if needed (`go build -o /tmp/sproot ./cmd/sproot` in the repo, or `make build`),
   then run `<sproot> validate --path <file>`. Outside the repo, ask the user to run
   `sproot validate --path <file>`.
2. `sproot validate` collects ALL violations at once. Map each error to the offending phase and
   propose the minimal fix (correct field name, required field, valid `install`/`mode` value,
   `phases` vs `targets` exclusivity, identity completeness).
3. Re-validate after edits.

### suggest: pick a module
Given a need ("I want ripgrep", "run a background service", "clone my dotfiles"), name the module,
give the minimal valid snippet, and note any prerequisite (e.g. cosign before a cosign-verified
`binary_release`; an `env` token for `gh_token`/`ssh_setup`).

## Reminders

- All phase fields are flat. `phases` and `targets` are mutually exclusive.
- The four `identity` fields are all required by `sproot validate`.
- No emdashes in generated output: use parens or commas (a sproot project convention).
