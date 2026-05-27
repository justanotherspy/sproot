# sproot plugin

A Claude Code plugin with two skills for working with [sproot](https://github.com/justanotherspy/sproot)
configs.

## Skills

- **`/sproot:script-convert`**: convert an existing setup/bootstrap bash script into a
  `sproot.yaml` plus any companion files it references, ready to drop into a sproot config repo.
  Maps `apt-get install` -> `apt`, `cargo install` -> `cargo_install`, heredocs -> `file_template`,
  `>> ~/.bashrc` -> `rc_block`, GitHub release downloads -> `binary_release`, and so on, falling
  back to `cmd` only when nothing structured fits. It validates its output with `sproot validate`
  and reports what needs human review.
- **`/sproot:author-config`**: generate a `sproot.yaml` from a description, explain an existing
  one phase by phase, interpret and fix `sproot validate` errors, or suggest the right module for
  a need.

## Install

```
/plugin marketplace add justanotherspy/claude-plugins
/plugin install sproot@justanotherspy
```

Then invoke a skill with `/sproot:script-convert <path-to-script>` or `/sproot:author-config`.

## Layout

```
plugins/sproot/
  .claude-plugin/plugin.json
  reference/
    module-map.md      # bash-idiom -> module decision table
    module-schema.md   # exact field reference for every module
  skills/
    script-convert/SKILL.md   # + examples/ golden fixtures
    author-config/SKILL.md
```

The `reference/` docs mirror `docs/modules.md` and `internal/config/schema.go` in the main repo;
those remain the canonical source of truth.
