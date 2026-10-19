# Changelog

## 1.0.0

The first release: Prometheus metrics of a Paper server through WorldAPI 3.0 or newer.

- **Performance:** TPS over three windows, tick times by percentile, CPU, and the JVM: heap and pools, garbage collection, threads, classes, buffers.
- **Worlds:** loaded chunks, entities and their most common types, block entities, players, the day count, time of day, weather, and the border.
- **Players:** who's online with ping, health, food, level, and idle time, and lifetime statistics from the game for everyone who has played: play time, deaths, kills, damage, blocks mined, items crafted, distance, jumps, fish, breeding, taming, trades, enchanting, diamonds, ancient debris, experience, advancements, first join, and last seen. `EXPORTER_PLAYERS=false` adds every player up instead.
- **Activity:** counters of the events WorldAPI records, other plugins' included, deaths by cause and killer, advancements by frame, and first-time players.
- **Many servers:** `/probe?target=<name>` reads the servers listed in `EXPORTER_TARGETS`.
- Statistics refresh a few at a time, so the exporter stays within WorldAPI's limit for anonymous clients.
- A Docker image for amd64 and arm64 at `ghcr.io/mairwunnx/minecraft-exporter`, and binaries for Linux, macOS, and Windows.
