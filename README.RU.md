# minecraft-exporter

<p align="center">
  <a href="https://github.com/MairwunNx/minecraft-exporter/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/MairwunNx/minecraft-exporter?label=release"></a>
  <a href="https://github.com/MairwunNx/minecraft-exporter/pkgs/container/minecraft-exporter"><img alt="Docker" src="https://img.shields.io/badge/docker-ghcr.io-2496ED?logo=docker&logoColor=white"></a>
  <a href="https://github.com/MairwunNx/minecraft-exporter/actions/workflows/ci.yml"><img alt="Build" src="https://img.shields.io/github/actions/workflow/status/MairwunNx/minecraft-exporter/ci.yml?branch=master&label=build"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/MairwunNx/minecraft-exporter"></a>
</p>

📊 **minecraft-exporter** превращает сервер Paper в метрики Prometheus: TPS и время тика, память, миры, статистику каждого игрока, смерти и достижения. Он читает [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi), поэтому серверу нужен только этот плагин. Добавьте [дашборд для Grafana](https://github.com/MairwunNx/minecraft-grafana) — и весь сервер будет на одной странице.

[English version](README.md)

![Дашборд Minecraft в Grafana](media/dashboard.png)

## Что вы получаете

- ⏱️ **Производительность:** TPS за одну, пять и пятнадцать минут, время тика по перцентилям, CPU, куча Java, сборка мусора, потоки.
- 🌍 **Миры:** загруженные чанки, сущности и самые частые их типы, блочные сущности, время суток, номер дня, погода.
- 🧑‍🤝‍🧑 **Игроки:** кто онлайн, с пингом, здоровьем и уровнем; когда каждый пришёл впервые и когда заходил последний раз; пожизненная статистика из самой игры: время в игре, смерти, убийства, добытые блоки, пройденный путь, алмазы, достижения и другое.
- 💀 **Активность:** входы, смерти по причинам и убийцам, достижения по рамкам, новые игроки и события, которыми другие плагины делятся через WorldAPI, например убийства боссов.
- 🔐 **Больше с токеном:** нагрузка, память и диск машины, предупреждения и ошибки консоли с указанием источника, плагины и их версии, что скопилось в каждом мире (вещи на земле, жители, самые загруженные чанки), где стоит каждый игрок и кто сколько пишет в чат.
- 🛰️ **Много серверов на один экспортер**, как у blackbox exporter.

Экспортер работает рядом с Prometheus, а не внутри игры: при каждом опросе он спрашивает WorldAPI, и серверу это стоит нескольких HTTP-запросов.

## Установка

Нужен Paper 26.3 или его форк с [WorldAPI](https://github.com/MairwunNx/minecraft-worldapi) 3.0 или новее. По умолчанию WorldAPI отдаёт анонимным клиентам всё, что нужно экспортеру, так что токен необязателен.

### Через Docker (рекомендуется)

```yaml
services:
  minecraft-exporter:
    image: ghcr.io/mairwunnx/minecraft-exporter:latest
    restart: unless-stopped
    environment:
      WORLDAPI_URL: http://minecraft:8110/api/v1
```

В [`examples/`](examples) лежит готовый стек: экспортер, Prometheus и Grafana с дашбордом. Скачайте дашборд, укажите `WORLDAPI_URL` и запустите стек. Grafana откроется на `http://localhost:3000` (admin / admin):

```sh
cd examples
curl -Lo grafana/dashboards/minecraft.json https://github.com/MairwunNx/minecraft-grafana/releases/latest/download/minecraft.json
WORLDAPI_URL=http://your-server:8110/api/v1 WORLDAPI_TOKEN=… docker compose up -d
```

Токен необязателен, но без него на дашборде пусты таблицы чата, событий и консоли. Прежде чем его задать, прочитайте раздел «Больше с токеном».

### Бинарником

Скачайте сборку для своей системы из [последнего релиза](https://github.com/MairwunNx/minecraft-exporter/releases/latest) и запустите:

```sh
WORLDAPI_URL=http://localhost:8110/api/v1 ./minecraft-exporter
```

Метрики появятся на `http://localhost:10059/metrics`.

## Prometheus

```yaml
scrape_configs:
  - job_name: minecraft
    static_configs:
      - targets: ["minecraft-exporter:10059"]
        labels: { instance: survival }
```

Метка `instance` — имя сервера на дашборде. Интервал в 15 секунд подходит хорошо.

### Несколько серверов

Перечислите серверы в YAML-файле и укажите его в `EXPORTER_TARGETS`:

```yaml
targets:
  survival:
    url: http://survival:8110/api/v1
    token_file: /run/secrets/survival   # или token: …, или ничего для анонимного чтения
  creative:
    url: http://creative:8110/api/v1
    players: false                      # заменяет EXPORTER_PLAYERS для этого сервера
```

Prometheus спрашивает каждый сервер по имени:

```yaml
scrape_configs:
  - job_name: minecraft
    metrics_path: /probe
    static_configs:
      - targets: [survival, creative]
    relabel_configs:
      - { source_labels: [__address__], target_label: __param_target }
      - { source_labels: [__param_target], target_label: instance }
      - { target_label: __address__, replacement: "minecraft-exporter:10059" }
```

## Настройки

Каждая настройка — переменная окружения или флаг.

| Переменная | Флаг | По умолчанию | Что делает |
|---|---|---|---|
| `WORLDAPI_URL` | `--worldapi.url` | | WorldAPI для `/metrics`, адрес заканчивается на `/api/v1` |
| `WORLDAPI_TOKEN` | `--worldapi.token` | | Токен WorldAPI; без него экспортер читает как посетитель |
| `WORLDAPI_TOKEN_FILE` | `--worldapi.token-file` | | Файл с токеном, для Docker secrets |
| `EXPORTER_TARGETS` | `--targets` | | Серверы для `/probe?target=<имя>` |
| `EXPORTER_PLAYERS` | `--players` | `true` | Метрики по игрокам с меткой `player`. При `false` все игроки складываются в одну серию, а подробности об игроках онлайн пропадают |
| `EXPORTER_STATS_INTERVAL` | `--stats-interval` | `60s` | Как часто перечитывать каждую статистику игроков |
| `EXPORTER_TIMEOUT` | `--timeout` | `10s` | Самый долгий допустимый опрос |
| `EXPORTER_LISTEN` | `--listen` | `:10059` | Где отдавать метрики |

## Больше с токеном

Без токена экспортер читает то, что WorldAPI показывает любому посетителю. Токен открывает больше — по одному праву за раз:

| Право | Что добавляет |
|---|---|
| `server:read` | Плагины и их версии, и что скопилось в каждом мире: вещи на земле, жители, питомцы, мобы с именами, самые загруженные чанки с воронками |
| `players:private` | Где стоит каждый игрок онлайн |
| `chat:read` | Сообщения чата, всего и по игрокам |
| `system:read` | Машина: нагрузка, память, своп, диск под папкой сервера, лимиты контейнера, открытые файлы, память процесса, Java |
| `logs:read` | Строки консоли по уровням, предупреждения и ошибки по источникам |
| `players:moderate` | Баны, операторы и вайтлист. Это право ещё и банит и кикает, так что не давайте его без нужды |

Токен со всем, что читает экспортер, и без права что-либо менять, задаётся в `config.yml` WorldAPI:

```yaml
auth:
  tokens:
    - name: monitoring
      token: "64 случайных шестнадцатеричных символа, например вывод openssl rand -hex 32"
      scopes: [viewer, "logs:read", "system:read"]
```

В группе `viewer` есть `server:read`, `players:read`, `players:private`, `events:read` и `chat:read`. Перезапустите сервер и передайте токен в `WORLDAPI_TOKEN` или `WORLDAPI_TOKEN_FILE`. Метрика `minecraft_exporter_endpoint_allowed` показывает, какие эндпоинты токен открывает; закрытый экспортер переспрашивает раз в 10 минут.

> [!WARNING]
> **Не выставляйте метрики с токеном наружу.** С токеном экспортер публикует то, что WorldAPI скрывает от посетителей: где стоит каждый игрок, кто сколько пишет в чат, предупреждения консоли, саму машину. Всё это читает любой, кто доберётся до `/metrics`, вашего Prometheus или Grafana с этими метриками. Источник данных Grafana с тем же токеном читает всё, что разрешает токен, включая сообщения чата. Держите экспортер, Prometheus и Grafana за логином или файрволом, никогда на публичном адресе или с анонимным доступом, и давайте токену только права на чтение.

Ещё с токеном действует лимит WorldAPI для токенов — по умолчанию 1200 запросов в минуту вместо 120 у посетителя, — и экспортер обновляет статистику большими партиями.

## Метрики

Все метрики начинаются с `minecraft_`. Счётчики событий и консоли считают с запуска экспортера, как принято у счётчиков Prometheus. Статистика игроков берётся из пожизненных счётчиков самой игры. Группам с правом в скобках нужен токен с этим правом.

| Группа | Метрики |
|---|---|
| Сервер | `up`, `info` (версии в метках), `start_time_seconds`, `players_online`, `players_max`, `players_known`, `tick_rate`, `tick_frozen`, `whitelist_enabled`, `datapacks_enabled` |
| Производительность | `tps{window}`, `tick_duration_seconds{quantile}`, `tick_duration_average_seconds`, `cpu_usage_ratio{scope}`, `cpu_cores` |
| JVM | `jvm_memory_{used,committed,max}_bytes{area}`, `jvm_memory_pool_*{pool}`, `jvm_gc_collections_total{gc}`, `jvm_gc_seconds_total{gc}`, `jvm_threads{state}`, `jvm_classes_loaded`, `jvm_buffer_pool_*{pool}` |
| Миры | `world_info{environment}`, `world_chunks_loaded`, `world_entities`, `world_entities_by_type{type}`, `world_block_entities`, `world_players`, `world_days_total`, `world_time_ticks`, `world_raining`, `world_thundering`, `world_border_size_blocks` |
| Игроки онлайн | `player_info{uuid,world,gamemode}`, `player_ping_seconds`, `player_health`, `player_food`, `player_level`, `player_idle_seconds` |
| Статистика игроков | `player_play_time_seconds_total`, `player_deaths_total`, `player_mob_kills_total`, `player_player_kills_total`, `player_damage_dealt_total`, `player_damage_taken_total`, `player_blocks_mined_total`, `player_items_crafted_total`, `player_distance_meters_total`, `player_jumps_total`, `player_fish_caught_total`, `player_animals_bred_total`, `player_animals_tamed_total`, `player_villager_trades_total`, `player_items_enchanted_total`, `player_diamonds_mined_total`, `player_ancient_debris_mined_total`, `player_experience_gained_total`, `player_advancements`, `player_first_join_timestamp_seconds`, `player_last_seen_timestamp_seconds` |
| Активность | `events_total{type}`, `deaths_total{cause,killer}`, `advancements_total{frame}`, `first_joins_total` |
| Чат (`chat:read`) | `player_chat_messages_total` и `player.chat` в `events_total` |
| Координаты (`players:private`) | `player_position_blocks{world,axis}` |
| Миры (`server:read`) | `world_dropped_items`, `world_dropped_item_count`, `world_villagers`, `world_named_mobs`, `world_tamed_mobs`, `chunk_entities{chunk}`, `chunk_block_entities{chunk}`, `chunk_hoppers{chunk}` для пяти самых загруженных чанков |
| Плагины (`server:read`) | `plugin_enabled{plugin,version}`, а при [Mo’Bosses](https://modrinth.com/plugin/mobosses) ещё `mobosses_alive`, `mobosses_spawned_total{type,class}`, `mobosses_killed_total{type,class}` |
| Машина (`system:read`) | `host_load_average{window}`, `host_cpus`, `host_memory_{total,available}_bytes`, `host_swap_{total,free}_bytes`, `disk_{total,free}_bytes`, `container_memory_{limit,used}_bytes`, `container_cpu_limit`, `process_open_files`, `process_max_open_files`, `process_resident_memory_bytes`, `java_info{name,vendor,version}` |
| Консоль (`logs:read`) | `log_messages_total{level}`, `log_problems_total{level,logger}` |
| Модерация (`players:moderate`) | `bans{kind}`, `operators`, `whitelist_players` |
| Экспортер | `exporter_scrape_duration_seconds`, `exporter_request_errors_total{target,endpoint}`, `exporter_endpoint_allowed{endpoint}`, `exporter_build_info` |

## Grafana

Дашборд [minecraft-grafana](https://github.com/MairwunNx/minecraft-grafana) рисует всё это, а ещё показывает таблицы чата и событий прямо из WorldAPI. Как его импортировать, написано в его README.

## На Moonlands

Moonlands, мой сервер для хардкорного выживания с друзьями, работает с этим экспортером и токеном для мониторинга за закрытой Grafana. Его сайт, [minecraft.mairwunnx.com](https://minecraft.mairwunnx.com), читает тот же WorldAPI.

[![Играйте на Moonlands: чистая ванилла, сложные мобы, ИИ, приваты. minecraft.mairwunnx.com](https://raw.githubusercontent.com/MairwunNx/minecraft-worldapi/master/media/og-adv-image.png)](https://minecraft.mairwunnx.com)

## Сборка

Go 1.27: `go test ./...` и `go build .`. Docker-образ собирается командой `docker build .`.

## Лицензия

[MIT](LICENSE).
