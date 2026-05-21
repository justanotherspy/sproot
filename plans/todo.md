# Todos

Items marked DONE are implemented and merged. Items with a phase reference are tracked in plans/sproot.md.

## Done (phases 8-12)

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

## Planned (tracked in plans/sproot.md)

- create a claude skill to convert a script into a sproot.yaml (Phase 14c)
- add a way to pass in a sproot.yaml file from the host instead of a git repo (Phase 13b)
- add an update to /.sprite/llm.txt and /.sprite/docs/agent-context.md (Phase 14a)
- have a sproot push command that pushes changes to all sproot-labeled sprites (Phase 13c)
- multi-target support in sproot.yaml with extends (Phase 13a)
- inter-sproot URL templating to pass connection strings between sprites (Phase 13a future direction)
- if we can better align our tool with the sdk, create a skill for sproot usage (Phase 14b)

## Planned (new, tracked as Phase 15)

- on config init inspect sprite config for an org and offer to select one automatically (Phase 15a)
- have a detailed explanation of what scopes are needed on the gh token (Phase 15b)
- what values are allowed for ram-mb and region? document them (Phase 15c)
- update required checks to have specific ones before merging so auto merge works (Phase 15d)
- update modules to better handle different cases (fewer cmd fallbacks) (Phase 15e)
- test out the release workflow (Phase 15f)
- consider a claude code review workflow (Phase 15g)
