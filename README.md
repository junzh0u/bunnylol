# bunnylol

A tiny URL redirect service in Go. Point your browser's default search
engine at it, and short queries become smart redirects — `yt cat videos`
goes straight to a YouTube search, a bare `gh` opens GitHub, and anything
unrecognized falls through to a default search.

Inspired by Facebook's internal [bunny1](https://github.com/ccheever/bunny1)-style
"smart keyword" bookmarks.

## How it works

The server exposes a single endpoint that reads the `?q=` query parameter
and issues a `307` redirect. A query resolves in this order:

1. **Empty query** → the default site's root.
2. **First token matches a keyword** (case-insensitive):
   - with more text → the keyword's URL, `%s` replaced by the rest of the query.
   - keyword alone → that site's root (e.g. `yt` → `https://youtube.com`).
3. **Full query matches a regex** (patterns tried in order) → that route's URL.
4. **No match** → the `Default` URL, with the whole query substituted in.

The `%s` placeholder is URL-encoded before substitution.

## Configuration

Config is JSON. Point to it with the `-config` flag or the `BUNNYLOL_CONFIG`
env var (falls back to `./config.json`). It is **hot-reloaded** — edits are
picked up within ~2s, no restart needed.

```json
{
  "Default": "https://www.google.com/search?q=%s",
  "Keywords": {
    "yt": "https://www.youtube.com/results?search_query=%s",
    "gh": "https://github.com/search?q=%s"
  },
  "Regexes": [
    { "pattern": "^\\d{5}$", "url": "https://www.google.com/search?q=zip+%s" }
  ]
}
```

- **`Default`** — fallback URL template with a `%s` placeholder (required).
- **`Keywords`** — map of prefix keyword → URL template.
- **`Regexes`** — ordered list of `{ "pattern", "url" }` routes, matched against
  the whole query.

## Running

```bash
just run              # port 8080, config from ~/.config/bunnylol/config.json
just run -port 9000   # custom port
just test             # run tests
just check            # go vet
just build            # build ./bunnylol binary
```

Or directly:

```bash
go run main.go -port 8080 -config ./config.json
```

### As your browser search engine

Add a custom search engine pointing at:

```
http://localhost:8080/?q=%s
```

## Deployment

Every push to `main` builds and publishes `ghcr.io/junzh0u/bunnylol:latest`
via GitHub Actions. `docker-compose.yml` runs that image:

```bash
docker compose up -d          # PORT defaults to 4000
```

The container reads its config from a **directory** bind-mount at
`/app/config` (a directory, not a single file, so atomic-save hot reloads
still work). `just deploy [host] [dir]` syncs the compose file to a remote
host, pulls the latest image, and restarts.

## License

[MIT](LICENSE)
