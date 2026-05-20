# Todos

- update docs to explain sproot.yaml validation
- create a claude skill to convert a script into a sproot.yaml
- update modules to better handle different cases like in justanotherspy/sprite as there are a lot of cmd fallbacks currently
- update config file to be a sytax like toml or yaml
- on config init we should inspect the sprite config for an org and offer to select one as the default automatically
- add a debug flag to show us what is going on at every step with debug logging through out
- fix this:

```
➜ sproot new test-sproot
- creating sprite test-sproot
- injecting sproot binary
- running setup in sprite test-sproot
exec: Exec format error
Error: exit status 1
Usage:
  sproot new <name> [flags]

Flags:
      --config-path string   path to config file within the repo (overrides host config)
      --cpus int             CPU count for the sprite (0 uses default)
      --dry-run              describe changes without executing them
      --force                re-run phases even if already complete
  -h, --help                 help for new
      --only string          run only phases matching this type
      --ram-mb int           RAM in MB for the sprite (0 uses default)
      --region string        region for the sprite (empty uses default)
```

- wrap the sprite console command so that on a sproot console we get console
- for sproot new we also get console and add a --no-console flag to new so we can disable that
- look for more sprite commands to wrap so that a user doesnt need sprite cli as well (should work without it only a sprite token is the requirement)
- If we run sproot and there is no config we set one up on the host
- make config setup interactive to ask for the details we need and validate at the end always. then make a non-interactive flag
- have a detailed explanation of what scopes are needed on the gh token at minimum (cloning repos for eg), and if we want to do an ssh key we need extra scopes
- what values are allowed for ram-mb and region? if left off we should defer to the api choices.
- we should have a sproot list which is a sprite list but only shows us sprites setup by sproot. we can add labels to all sprites that show that it was made by sproot
- look at findings.md and address the discrepencies
- add a way to pass in a sproot.yaml file from the host to the sprite instead of using a git repo as config source. maybe make config block in host config take in optional pointers to repo source or local path
- review the sprite-go sdk and remove things from sproot that are not supported
- test out checkpointing and wrap those commands too, make sproot.yaml decide if we should checkpoint after setup. doing it before is pointless as the checkpoint will just be what is in the base image already
- make sure that our state is readable from the host easily and shows a summary to the user of what is setup and if there is any out standing module phases.
- if we can better align our tool with the sdk, improve our sproot yaml and modules and have good documentation we should have a skill we can install for sproot usage
