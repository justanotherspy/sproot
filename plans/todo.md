# Todos

1. update docs to explain sproot.yaml validation
2. create a claude skill to convert a script into a sproot.yaml
3. update modules to better handle different cases like in justanotherspy/sprite as there are a lot of cmd fallbacks currently
4. update config file to be a sytax like toml or yaml
5. on config init we should inspect the sprite config for an org and offer to select one as the default automatically
6. add a debug flag to show us what is going on at every step with debug logging through out
7. fix this: 
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
8. wrap the sprite console command so that on a sproot console we get console
9. for sproot new we also get console and add a --no-console flag to new so we can disable that
10. look for more sprite commands to wrap so that a user doesnt need sprite cli as well (should work without it only a sprite token is the requirement)
11. If we run sproot and there is no config we set one up on the host
12. make config setup interactive to ask for the details we need and validate at the end always. then make a non-interactive flag
13. have a detailed explanation of what scopes are needed on the gh token at minimum (cloning repos for eg), and if we want to do an ssh key we need extra scopes
14. what values are allowed for ram-mb and region? if left off we should defer to the api choices.
15. we should have a sproot list which is a sprite list but only shows us sprites setup by sproot. we can add labels to all sprites that show that it was made by sproot
