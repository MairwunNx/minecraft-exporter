# minecraft-exporter: notes for AI agents

A Prometheus exporter that reads a Minecraft server through WorldAPI (MairwunNx/minecraft-worldapi) and serves its metrics. README.md explains it for humans; read it first. The dashboard that draws these metrics lives in MairwunNx/minecraft-grafana (`~/c/personal/minecraft-grafana`). Moonlands, the owner's server, runs the exporter from its `compose.yaml` in `~/c/personal/minecraft` (MairwunNx/minecraft).

## Talking to the owner

- Reply in Russian, address the owner as «вы». Quotes «ёлочки», nested „лапки“, an em dash with spaces « — ».
- Code, comments, commit messages, metric descriptions, and README.md are in English: curly quotes, an em dash without spaces or an en dash with spaces, never a hyphen as a dash. README.RU.md follows the Russian rules.

## How it fits together

| Path | What it does |
|---|---|
| `main.go` | Flags and environment variables, `/metrics`, `/probe`, the exporter's own metrics |
| `internal/config/` | The targets file behind `/probe`, URL and token checks |
| `internal/worldapi/` | The few WorldAPI endpoints the exporter reads, and their JSON |
| `internal/collector/target.go` | One server: a scrape reads WorldAPI in parallel and sends constant metrics |
| `internal/collector/server.go` | Server, performance, JVM, worlds, online players |
| `internal/collector/stats.go` | Player statistics from leaderboards, cached and refreshed a batch per scrape |
| `internal/collector/events.go` | Counters fed by `GET /events`, counted from the exporter's start |
| `internal/collector/private.go` | What a token opens: the host, the console, plugins, moderation, Mo’Bosses, what crowds each world |

## Rules that keep it working

- Metric names start with `minecraft_`, follow Prometheus conventions (`_total` for counters, base units: seconds, bytes, meters), and never change once released: dashboards and alerts depend on them. A rename is a major version.
- The `minecraft-grafana` dashboard uses these metrics. When one is added, renamed, or changes meaning, change the dashboard in the same go, and the metrics table in both READMEs.
- Labels stay bounded: worlds, entity types (the top ten WorldAPI names), players, event types, death causes. Never a label from free text such as a chat message or a death message.
- A scrape stays within WorldAPI's limit for anonymous clients (120 requests a minute, bursts of 30): leaderboards refresh `statsBatch` at a time. Count the requests when adding an endpoint.
- A failed request drops only its own metrics and counts in `minecraft_exporter_request_errors_total`. A refusal (401, 403) is not an error: `done()` closes the endpoint for 10 minutes, `allowed()` skips it, and `minecraft_exporter_endpoint_allowed` shows it. `minecraft_up` follows `GET /server` alone.
- Data behind a permission goes through `allowed()` and `done()` with the endpoint's name, and the README's “More with a token” table and metrics table name the permission. Slow data (once a minute or rarer) joins the batched queue in `privateItems()`.
- Metrics from a token can reveal positions, chat, and the machine: the README's warning about keeping them private stays.
- Never print or log `WORLDAPI_TOKEN`. The owner's read-only WorldAPI token lives in `~/.secrets/cloud.env` as `WORLDAPI_TOKEN`.

## Checking a change

1. `gofmt -l .` prints nothing, `go vet ./...` and `go test -race ./...` pass. Tests use a fake WorldAPI and never touch the network.
2. Against Moonlands' WorldAPI, read-only: `WORLDAPI_URL=https://minecraft.mairwunnx.com/worldapi/api/v1 EXPORTER_LISTEN=127.0.0.1:10059 go run .`, then `curl -s 127.0.0.1:10059/metrics` twice, 15 seconds apart. Every group from the README shows up, and `minecraft_exporter_request_errors_total` stays absent.
3. A change to what the dashboard shows: run `examples/` with the image built locally (`docker build -t minecraft-exporter:test .`) and look at the dashboard, as minecraft-grafana's CLAUDE.md says.

## Autopilot

`.github/workflows/autopilot.yml` runs Claude every Monday at 03:00 UTC: `/update-deps`, a fix for whatever it breaks, a push to `master`, and the “Release” workflow when the exporter changed. It needs the `CLAUDE_CODE_OAUTH_TOKEN` secret. “Autopilot” in `~/c/personal/minecraft/CLAUDE.md` explains the family's runs. A session with the owner still releases only when the owner asks.

## Versions

Where the versions live, for `/update-deps` and `/update-minecraft` (both in `.claude/skills/`, the same files in every project of the family). This project has no Minecraft version to move, so `/update-minecraft` stops at once: it follows WorldAPI's endpoints, not the game.

- Go: `toolchain` in `go.mod` and the `golang` image in `Dockerfile`, the same version. The `go` line stays at the oldest Go the code needs.
- Modules: `go.mod` and `go.sum`.
- Images: `Dockerfile` (`golang`, `gcr.io/distroless/static-debian13:nonroot`), and Prometheus and Grafana in `examples/compose.yaml`, matched to Moonlands' `compose.yaml`.
- GitHub Actions in `.github/workflows/`.

## Docs and releases

- README.md and README.RU.md are for people who run the exporter: what it gives, install, Prometheus, settings, metrics. Change both together. Internals stay here.
- After a change that reaches what the project ships, add it to CHANGELOG.md in the same push: under the topmost “## X.Y.Z” section while that version isn't released (`gh release view vX.Y.Z` fails), or under a new section with the next version: patch for fixes, minor for features, major for breaking changes (a renamed or removed metric).
- Release: `gh workflow run release.yml -R MairwunNx/minecraft-exporter -f version=X.Y.Z`. It tests, builds binaries for Linux, macOS, and Windows on amd64 and arm64 with `checksums.txt`, pushes `ghcr.io/mairwunnx/minecraft-exporter` for amd64 and arm64 as `X.Y.Z`, `X.Y`, `X`, and `latest`, and publishes the GitHub release. Its `deploy` job then starts `ship-plugin.yml` in MairwunNx/minecraft (the `FAMILY_ACTIONS_TOKEN` secret), which moves Moonlands to the new image. Afterwards check the release page and `docker pull ghcr.io/mairwunnx/minecraft-exporter:X.Y.Z`.
- The container package must stay public (GitHub → Packages → minecraft-exporter → Package settings), or `docker pull` asks for a login.
- Port 10059 is the exporter's entry in Prometheus' list of default ports (the “Default port allocations” page of the Prometheus wiki).

## Commits

Short one-line imperative English messages, one logical change per commit: “Count deaths by killer”.
