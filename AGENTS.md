# AGENTS.md

bunnylol is a URL redirect service in one Go file (`main.go`, tests in `server_test.go`). [README.md](README.md) covers how a query resolves, the config format, running and deployment; this file adds what an agent needs beyond it.

## Config

- `config.json` is not in this repository. The service reads `~/.config/bunnylol/config.json`, which the owner's dotfiles manage on every host: change keywords and routes there. `just run` reads the same file.
- The service reloads the file within about 2 seconds of a change, so a config change needs no restart or deploy.
- A regex route takes the whole query, and whatever no route catches goes to `Default`. Before changing a pattern, try it on queries it must catch and on everyday searches it must not.

## Workflow

- Run `just check` (go vet) and `just test` before committing; `just fix` runs go fmt.
- Every push to `main` publishes `ghcr.io/junzh0u/bunnylol:latest` through GitHub Actions. `just deploy` falls through to the global deploy recipe (`set fallback`): it syncs only `compose.yaml` to the host in `DEPLOY_HOST` (the gitignored `.envrc`), pulls the image and restarts. The host never checks out this repository.
