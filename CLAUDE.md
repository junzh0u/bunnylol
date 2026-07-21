# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bunnylol is a URL redirect service written in Go. It takes search queries and redirects them to configured search engines based on keyword prefixes or regex pattern matches.

## Build and Run Commands

```bash
just run              # Run the server (default port 8080)
just run -port 9000   # Run with custom port
just test             # Run tests
just test -run TestMatchEngine  # Run a single test
just check            # Lint (go vet)
just fix              # Auto-format (go fmt)
just build            # Build binary
```

## Architecture

The application is a single-file Go HTTP server (`main.go`) with three main components:

1. **Config loading** (`loadConfig`): Reads JSON config with default URL, keyword mappings, and regex patterns. Config is hot-reloaded on file change (polling every 2s) via `watchConfig`.
2. **Query matching** (`matchEngine`): Matches incoming queries against keywords (prefix match) or regexes, returns appropriate search engine URL
3. **HTTP handler** (`makeHandler`): Extracts `?q=` parameter, runs matching, performs 307 redirect

**Query resolution order:**
1. Empty query: redirects to default site root
2. First token checked against `Keywords` map (case-insensitive)
   - With query: redirects to search URL with `%s` replaced
   - Without query: redirects to site root (e.g., `yt` → `youtube.com`)
3. Full query matched against `Regexes` patterns (in order)
4. Falls back to `Default` URL

Note: The `%s` placeholder is URL-encoded before substitution.

**Config location**: `config.json` is **not** in this repo. It lives at `~/.config/bunnylol/config.json`, tracked in the dotfiles repo (`~/.dotfiles/.config/bunnylol/`) and kept in sync across hosts by dotfiles. Edit it there. The config path is set via the `BUNNYLOL_CONFIG` env var (falls back to `./config.json` if unset); `-config` flag overrides both. `just run` points `BUNNYLOL_CONFIG` at the `~/.config` copy.

**Config format** (`config.json`):
- `Default`: Fallback URL template with `%s` placeholder
- `Keywords`: Map of prefix keywords to URL templates
- `Regexes`: Ordered array of `{"pattern": "...", "url": "..."}` objects

**Deployment** (`just deploy [host] [dir]`, defaults `gateway`/`bunnylol`): The remote never checks out this repo. GitHub Actions (`.github/workflows/docker.yml`) builds and pushes `ghcr.io/junzh0u/bunnylol:latest` on every push to `main`. `just deploy` scps only `docker-compose.yml` to the host, then pulls the image and restarts via `docker compose`. Config reaches the host separately via dotfiles (`git pull` in `~/.dotfiles`), so gateway must have dotfiles set up.

**Docker**: `docker-compose.yml` runs the published image. `PORT` (compose var, defaults `4000`) sets the listen port; `BUNNYLOL_CONFIG=/app/config/config.json` points at the mount. The host's `~/.config/bunnylol` dir is bind-mounted at `/app/config:ro` — a directory mount (not a single file) so the config watcher sees hot-reload updates when dotfiles rewrites the file.
