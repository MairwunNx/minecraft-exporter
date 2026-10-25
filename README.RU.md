# minecraft-exporter

[![Release](https://img.shields.io/github/v/release/MairwunNx/minecraft-exporter?label=release)](https://github.com/MairwunNx/minecraft-exporter/releases/latest) [![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker&logoColor=white)](https://github.com/MairwunNx/minecraft-exporter/pkgs/container/minecraft-exporter) [![Build](https://img.shields.io/github/actions/workflow/status/MairwunNx/minecraft-exporter/ci.yml?branch=master&label=build)](https://github.com/MairwunNx/minecraft-exporter/actions/workflows/ci.yml) [![License](https://img.shields.io/github/license/MairwunNx/minecraft-exporter)](LICENSE)

📊 **minecraft-exporter** превращает сервер Minecraft на Paper, Fabric или NeoForge и даже мир, где вы играете в одиночку, в метрики Prometheus: TPS и время тика, память, миры, статистику каждого игрока, смерти и достижения. Он читает [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi), поэтому серверу нужен только этот плагин или мод. Добавьте [дашборд для Grafana](https://github.com/MairwunNx/minecraft-grafana) — и весь сервер будет на одной странице.

[English version](README.md)

![Дашборд Minecraft в Grafana: сервер, производительность и машина](https://raw.githubusercontent.com/MairwunNx/minecraft-grafana/master/media/overview.png)

![Игроки на дашборде: кто онлайн и статистика каждого](https://raw.githubusercontent.com/MairwunNx/minecraft-grafana/master/media/players.png)

## Что вы получаете

- ⏱️ **Производительность:** TPS за одну, пять и пятнадцать минут, время тика по перцентилям, CPU, куча Java, сборка мусора, потоки.
- 🌍 **Миры:** загруженные чанки, сущности и самые частые их типы, блочные сущности, время суток, номер дня, погода.
- 🧑‍🤝‍🧑 **Игроки:** кто онлайн, с пингом, здоровьем и уровнем; когда каждый пришёл впервые и когда заходил последний раз; пожизненная статистика из самой игры: время в игре, смерти, убийства, добытые блоки, пройденный путь, алмазы, достижения и другое.
- 💀 **Активность:** входы, смерти по причинам и убийцам, достижения по рамкам, новые игроки и события, которыми другие плагины делятся через WorldAPI.
- 🔐 **Больше с токеном:** нагрузка, память и диск машины, предупреждения и ошибки консоли, плагины и их версии, что скопилось в каждом мире, где стоит каждый игрок и кто сколько пишет в чат.
- 🛰️ **Много серверов на один экспортер**, как у blackbox exporter.
- 🪶 **Лёгкий для сервера:** работает рядом с Prometheus, а не внутри игры, и стоит серверу нескольких HTTP-запросов за опрос.

## Начало работы

Нужен [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi) 3.0 или новее на Paper 26.3 либо 4.2.0 или новее на Fabric или NeoForge. Играете один? Шаги — на странице [Singleplayer](https://github.com/MairwunNx/minecraft-exporter/wiki/Singleplayer).

1. Запустите экспортер с адресом вашего WorldAPI:

   ```sh
   docker run -d -p 10059:10059 -e WORLDAPI_URL=http://your-server:8110/api/v1 ghcr.io/mairwunnx/minecraft-exporter
   ```

2. Направьте Prometheus на `your-exporter:10059` и импортируйте в Grafana дашборд [25941](https://grafana.com/grafana/dashboards/25941).
3. В [Installation](https://github.com/MairwunNx/minecraft-exporter/wiki/Installation) — Compose, бинарник и готовый стек с Prometheus и Grafana.

## Документация

Руководство живёт в [вики](https://github.com/MairwunNx/minecraft-exporter/wiki), на английском.

Для владельцев серверов:

- [Installation](https://github.com/MairwunNx/minecraft-exporter/wiki/Installation)
- [Configuration](https://github.com/MairwunNx/minecraft-exporter/wiki/Configuration)
- [WorldAPI token](https://github.com/MairwunNx/minecraft-exporter/wiki/WorldAPI-token)
- [Prometheus](https://github.com/MairwunNx/minecraft-exporter/wiki/Prometheus)
- [Grafana](https://github.com/MairwunNx/minecraft-exporter/wiki/Grafana)
- [Metrics](https://github.com/MairwunNx/minecraft-exporter/wiki/Metrics)
- [Troubleshooting](https://github.com/MairwunNx/minecraft-exporter/wiki/Troubleshooting)

Для разработчиков:

- [Building from source](https://github.com/MairwunNx/minecraft-exporter/wiki/Building-from-source)
- [Architecture](https://github.com/MairwunNx/minecraft-exporter/wiki/Architecture)
- [Development](https://github.com/MairwunNx/minecraft-exporter/wiki/Development)
- [Releasing](https://github.com/MairwunNx/minecraft-exporter/wiki/Releasing)

## Связанные проекты

- **[Moonlands](https://minecraft.mairwunnx.com)** — сервер для хардкорного выживания, на котором всё это работает.
- **[WorldAPI](https://github.com/MairwunNx/minecraft-worldapi)** — REST и WebSocket API к миру, игрокам и статистике сервера на Paper, Fabric или NeoForge.
- **[Minecraft MCP](https://github.com/MairwunNx/minecraft-mcp)** — даёт ИИ-клиенту читать сервер и управлять им через WorldAPI.
- **[Allay](https://github.com/allay-ai/minecraft)** — ИИ-помощник в игре: спросите через `/ask` прямо в чате.
- **[Allay Server](https://github.com/allay-ai/server)** — бэкенд Allay.
- **[Mo’Joins](https://github.com/MairwunNx/mojoins)** — сообщения о входе и выходе на языке каждого игрока.
- **[Mo’Afks](https://github.com/MairwunNx/moafks)** — безопасный режим AFK, который не мешает пропустить ночь.
- **[Mo’Bosses](https://github.com/MairwunNx/mobosses)** — монстры, которые появляются боссами с классами, способностями и добычей.
- **[minecraft-grafana](https://github.com/MairwunNx/minecraft-grafana)** — дашборд Grafana для WorldAPI.
