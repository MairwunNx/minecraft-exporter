# Changelog

## 2.0.0

- **Mo’Bosses is gone.** `minecraft_mobosses_alive`, `minecraft_mobosses_spawned_total`, and `minecraft_mobosses_killed_total` no longer exist, and the exporter no longer reads `/extensions/mobosses/stats`. A plugin's events still count in `minecraft_events_total`, like any other plugin's.
- `examples/mock/` fakes a day of a busy server, with no Minecraft and no exporter, for the dashboard's screenshots.

## 1.1.0

- 🔐 **More with a token.** Each WorldAPI permission opens its own metrics: the host's load, memory, disk, and open files (`system:read`); console lines by level, and warnings and errors by who logged them (`logs:read`); plugins and their versions, and what crowds each world: items on the ground, villagers, pets, named mobs, the five busiest chunks with their hoppers (`server:read`); where each online player stands (`players:private`); chat messages per player (`chat:read`); bans, operators, and the whitelist (`players:moderate`).
- **Mo’Bosses:** bosses alive, and spawned and killed by type and class, when the plugin shares its data through WorldAPI.
- **A missing permission is no longer an error.** The exporter leaves an endpoint its token can't read for 10 minutes, and `minecraft_exporter_endpoint_allowed` shows which ones are open; `minecraft_exporter_request_errors_total` counts real failures only.
- With a token, statistics refresh 12 at a time instead of 6.
- The README warns to keep the metrics of a token private and shows a read-only token for WorldAPI's config.

## 1.0.0

The first release: Prometheus metrics of a Paper server through WorldAPI 3.0 or newer.

- **Performance:** TPS over three windows, tick times by percentile, CPU, and the JVM: heap and pools, garbage collection, threads, classes, buffers.
- **Worlds:** loaded chunks, entities and their most common types, block entities, players, the day count, time of day, weather, and the border.
- **Players:** who's online with ping, health, food, level, and idle time, and lifetime statistics from the game for everyone who has played: play time, deaths, kills, damage, blocks mined, items crafted, distance, jumps, fish, breeding, taming, trades, enchanting, diamonds, ancient debris, experience, advancements, first join, and last seen. `EXPORTER_PLAYERS=false` adds every player up instead.
- **Activity:** counters of the events WorldAPI records, other plugins' included, deaths by cause and killer, advancements by frame, and first-time players.
- **Many servers:** `/probe?target=<name>` reads the servers listed in `EXPORTER_TARGETS`.
- Statistics refresh a few at a time, so the exporter stays within WorldAPI's limit for anonymous clients.
- A Docker image for amd64 and arm64 at `ghcr.io/mairwunnx/minecraft-exporter`, and binaries for Linux, macOS, and Windows.
