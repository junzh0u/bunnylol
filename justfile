# Run the server (default port 8080, config from ~/.config/bunnylol)
run *args:
    BUNNYLOL_CONFIG="${XDG_CONFIG_HOME:-$HOME/.config}/bunnylol/config.json" go run main.go {{args}}

# Run all tests
test *args:
    go test -v {{args}}

# Lint and vet (non-destructive)
check:
    go vet ./...
    @echo "All checks passed."

# Auto-fix (go fmt)
fix:
    go fmt ./...

# Build the binary
build:
    go build -o bunnylol

# Deploy to remote host: sync compose file, pull latest image, restart.
# Config lives in ~/.config/bunnylol on the host, kept in sync by dotfiles.
deploy host="gateway" dir="bunnylol":
    scp docker-compose.yml {{ host }}:{{ dir }}/docker-compose.yml
    ssh {{ host }} "cd {{ dir }} && docker compose pull --quiet && docker compose up -d --remove-orphans && docker image prune -f"
    ssh {{ host }} "cd {{ dir }} && docker compose ps"
