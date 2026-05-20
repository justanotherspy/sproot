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
- update required checks to have specific ones before merging so that auto merge works better
- consider a claude code review workflow
- test out the release workflow
- add an update to the /.sprite/llm.txt and /.sprite/docs/agent-context.md files info about the changes to the environment based on what was installed by sproot
- have a sproot push command that can push a change to all your sprites setup by sproot (check via labels). so lets say we have 10 sproots already setup, and we want to add a new skill we made we can do sproot push <some flags or files that indicate the change> or maybe we have updated our sproot.yaml file either in the config repo or in the host machine and we push that to all the sproots (wake them up and push change and run setup again). this should detect the module phase not installed and then do the setup, or if the config is in a repo we pull the latest files from the remote and then do the setup script. we can do this in parallel to all sproots (doing a checkpoint before updates). or maybe a sproot update command for that which does it for us. and push would be more specific changes and we can apply it to specific sproots.
- i think sproot should look at the sproot.yaml before spawning a sprite to set it up, it can validate it but also make sure it exists. we can then evolve our sproot.yaml to be able to have more than one sproot config! so it could have a sproot-a which installs x/y/z and sproot-b block that installs slightly different things. we can specify which sproot target we want to setup on the command or we can do all of the sproots. maybe then we can even do something neat where if we want one sproot to give its URL to another sproot we do the first sproot and then template the url into the next sproots setup to just hand over the info from the one sproot to the other during setup stage. so like if one sproot is setup to run postgres and we setup a url and network port forward situation then we can hand over the connection string to another sproot in the same config so that it spins up the web server for eg and has the url ready to go from the setup.
