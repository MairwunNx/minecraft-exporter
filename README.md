# minecraft-exporter

[![Release](https://img.shields.io/github/v/release/MairwunNx/minecraft-exporter?label=release)](https://github.com/MairwunNx/minecraft-exporter/releases/latest) [![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker&logoColor=white)](https://github.com/MairwunNx/minecraft-exporter/pkgs/container/minecraft-exporter) [![Build](https://img.shields.io/github/actions/workflow/status/MairwunNx/minecraft-exporter/ci.yml?branch=master&label=build)](https://github.com/MairwunNx/minecraft-exporter/actions/workflows/ci.yml) [![License](https://img.shields.io/github/license/MairwunNx/minecraft-exporter)](LICENSE)

📊 **minecraft-exporter** turns your Paper server into Prometheus metrics: TPS and tick times, memory, worlds, every player's statistics, deaths, and advancements. It reads [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi), so the server needs that one plugin and nothing else. Pair it with the [Grafana dashboard](https://github.com/MairwunNx/minecraft-grafana) and see the whole server on one page.

[Русская версия](README.RU.md)

![The Minecraft dashboard in Grafana: server, performance, and host](https://raw.githubusercontent.com/MairwunNx/minecraft-grafana/master/media/overview.png)

![Players on the dashboard: who's online and every player's statistics](https://raw.githubusercontent.com/MairwunNx/minecraft-grafana/master/media/players.png)

## What you get

- ⏱️ **Performance:** TPS over one, five, and fifteen minutes, tick times by percentile, CPU, the Java heap, garbage collection, threads.
- 🌍 **Worlds:** loaded chunks, entities and the most common types, block entities, time of day, the day count, weather.
- 🧑‍🤝‍🧑 **Players:** who's online with ping, health, and level, when each player joined and was last seen, and lifetime statistics from the game itself: play time, deaths, kills, blocks mined, distance, diamonds, advancements, and more.
- 💀 **Activity:** joins, deaths by cause and killer, advancements by frame, first-time players, and events other plugins share through WorldAPI.
- 🔐 **More with a token:** the host's load, memory, and disk, the console's warnings and errors, plugins and their versions, what crowds each world, where each player stands, and who chats how much.
- 🛰️ **Many servers from one exporter**, the way the blackbox exporter does it.
- 🪶 **Light on the server:** it runs next to Prometheus, not inside the game, and costs a few HTTP requests per scrape.

## Get started

You need Paper 26.3 with [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi) 3.0 or newer.

1. Run the exporter with your WorldAPI's address:

   ```sh
   docker run -d -p 10059:10059 -e WORLDAPI_URL=http://your-server:8110/api/v1 ghcr.io/mairwunnx/minecraft-exporter
   ```

2. Point Prometheus at `your-exporter:10059`, and import dashboard [25941](https://grafana.com/grafana/dashboards/25941) into Grafana.
3. Read [Installation](https://github.com/MairwunNx/minecraft-exporter/wiki/Installation) for Compose, a binary, or the whole stack with Prometheus and Grafana ready.

## Documentation

The [wiki](https://github.com/MairwunNx/minecraft-exporter/wiki) is the manual.

For server owners:

- [Installation](https://github.com/MairwunNx/minecraft-exporter/wiki/Installation)
- [Configuration](https://github.com/MairwunNx/minecraft-exporter/wiki/Configuration)
- [WorldAPI token](https://github.com/MairwunNx/minecraft-exporter/wiki/WorldAPI-token)
- [Prometheus](https://github.com/MairwunNx/minecraft-exporter/wiki/Prometheus)
- [Grafana](https://github.com/MairwunNx/minecraft-exporter/wiki/Grafana)
- [Metrics](https://github.com/MairwunNx/minecraft-exporter/wiki/Metrics)
- [Troubleshooting](https://github.com/MairwunNx/minecraft-exporter/wiki/Troubleshooting)

For developers:

- [Building from source](https://github.com/MairwunNx/minecraft-exporter/wiki/Building-from-source)
- [Architecture](https://github.com/MairwunNx/minecraft-exporter/wiki/Architecture)
- [Development](https://github.com/MairwunNx/minecraft-exporter/wiki/Development)
- [Releasing](https://github.com/MairwunNx/minecraft-exporter/wiki/Releasing)

## Related projects

- **[Moonlands](https://minecraft.mairwunnx.com)**—the hard-survival server where all of these run.
- **[WorldAPI](https://github.com/MairwunNx/minecraft-worldapi)**—REST and WebSocket API to a Paper server's world, players, and stats.
- **[Minecraft MCP](https://github.com/MairwunNx/minecraft-mcp)**—lets an AI client read and run a server through WorldAPI.
- **[Allay](https://github.com/allay-ai/minecraft)**—an AI helper in the game: ask with `/ask`, right in the chat.
- **[Allay Server](https://github.com/allay-ai/server)**—the backend behind Allay.
- **[Mo’Joins](https://github.com/MairwunNx/mojoins)**—join and quit messages in each player's language.
- **[Mo’Afks](https://github.com/MairwunNx/moafks)**—a safe AFK mode that doesn't block the night.
- **[Mo’Bosses](https://github.com/MairwunNx/mobosses)**—monsters that spawn as bosses with classes, abilities, and loot.
- **[minecraft-grafana](https://github.com/MairwunNx/minecraft-grafana)**—the Grafana dashboard for WorldAPI.
