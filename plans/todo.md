# Todos

Items marked DONE are implemented and merged. Items with a phase reference are tracked in plans/sproot.md.

## Done (phases 8-16)

- update docs to explain sproot.yaml validation (DONE: Phase 8)
- update config file format (DONE: YAML, decided in Phase 1)
- add a debug flag (DONE: Phase 11g)
- fix exec: Exec format error when running setup (DONE: Phase 10)
- wrap the sprite console command (DONE: Phase 11d)
- for sproot new we also get console with --no-console flag (DONE: Phase 11c)
- look for more sprite commands to wrap: exec, upgrade, checkpoint, checkpoints, restore (DONE: Phase 12a)
- if we run sproot and there is no config we set one up on the host (DONE: Phase 11f)
- make config setup interactive with --non-interactive flag (DONE: Phase 11a)
- we should have sproot list which shows only sproot-labeled sprites (DONE: Phase 11e)
- look at findings.md and address the discrepancies (DONE: Phases 8 and 9)
- review the sprite-go sdk and remove unsupported things (DONE: Phase 12b)
- test out checkpointing and wrap those commands (DONE: Phase 12a)
- make checkpoint_after_setup in sproot.yaml (DONE: Phase 12c)
- make sure state is readable from the host (DONE: Phase 12d)
- validate sproot.yaml before spawning a sprite (DONE: Phase 11h)
- multi-target support in sproot.yaml with extends (DONE: Phase 13a)
- add a way to pass in a sproot.yaml file from the host instead of a git repo (DONE: Phase 13b)
- have a sproot push command that pushes changes to all sproot-labeled sprites (DONE: Phase 13c)
- multi-phase and multi-target CI jobs, --skip-verify flag, smart verify, make e2e (DONE: Phase 16)
- HostConfig field rename to sproot_config_* prefix (DONE: Phase 16j)
- remove unsupported flags (--ram-mb, --cpus, --region, --storage-gb) (DONE: Phase 15c/15h)

## Done (Phase 17 - code quality and bug fixes)

- fix sproot push silently dropping env block and GH_TOKEN forwarding - CRITICAL (DONE: Phase 17a)
- add HTTP timeouts to deleteGHKey and postGHKey (DONE: Phase 17b)
- consolidate sprootLabel and labelBase duplicate constants (DONE: Phase 17c)
- ConfigSHA format optimization: fmt.Sprintf("%x", h[:6]) (DONE: Phase 17d)
- omit empty labelTarget from Labels() (DONE: Phase 17e)
- add --only flag to README push row (already done, skipped)
- create MIGRATION.md with known cmd workarounds (DONE: Phase 17g)
- mark plans/findings.md as superseded (already done, skipped)

## Planned (Phase 18 - intelligence)

- add an update to /.sprite/llm.txt and /.sprite/docs/agent-context.md after setup (Phase 18a)
- have a detailed explanation of what scopes are needed on the gh token (Phase 18b)
- on config init inspect sprite config for an org and offer to select one automatically (Phase 18c)
- test out the release workflow end-to-end (Phase 18d)

## Deferred

- inter-sproot URL templating to pass connection strings between sprites (Phase 13a future direction)
- create a claude skill to convert a script into a sproot.yaml (Phase 14c)
- if we can better align our tool with the sdk, create a skill for sproot usage (Phase 14b)
- update required checks to have specific CI jobs required before merging so auto-merge works (Phase 15d, GitHub settings change)
- consider a claude code review workflow (Phase 15g)
- currentConfigSHA re-clones the git repo on every sproot outdated call (Phase 17i, deferred until users complain)
- RunNew clones the git config repo twice end-to-end (Phase 17j, deferred until startup time is a complaint)
