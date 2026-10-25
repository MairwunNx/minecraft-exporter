"""Fake a day of a busy Minecraft server for the dashboard's screenshots.

Writes metrics.om (OpenMetrics for `promtool tsdb create-blocks-from openmetrics`), and events.json and
logs.json for fake_worldapi.py, all from one simulation, so the tables match the graphs.
"""
import json
import math
import os
import random
import time
from collections import defaultdict

OUT = os.environ.get("MOCK_OUT", os.path.dirname(os.path.abspath(__file__)))
STEP = 30
NOW = int(time.time()) // STEP * STEP
START = NOW - 26 * 3600
END = NOW + 3 * 3600  # a little ahead, so instant queries find data while the screenshots run
TZ = 3 * 3600  # the players' evening, in UTC+3
rng = random.Random(26)
GiB = 1024 ** 3
MiB = 1024 ** 2

BASE = {"job": "minecraft", "instance": "survival"}
series = defaultdict(list)  # (name, labels tuple) -> [(t, v)]


def put(t, metric, value, **labels):
    series[(metric, tuple(sorted(labels.items())))].append((t, value))


# ── Players and their sessions.
PLAYERS = {
    # name: (lifetime hours played, evening keenness, ping ms, home world bias)
    "frostbyte": (412, 0.9, 38), "MossyCobble": (388, 0.8, 54), "Lumen_": (301, 0.7, 47),
    "RedstoneRita": (276, 0.85, 62), "Starling": (190, 0.6, 71), "Wanderer42": (144, 0.5, 118),
    "obsidian_owl": (96, 0.55, 44), "kelp_king": (0.0, 0.7, 83),
}
DORMANT = {"pixel_fern": 52, "Tidewalker": 31, "copperbell": 12}
NEW_PLAYER = "kelp_king"


def local_hour(t):
    return ((t + TZ) % 86400) / 3600


def sessions_for(name, keen):
    out, t = [], START
    while t < END:
        h = local_hour(t)
        evening = 1.0 if 17 <= h or h < 1 else (0.45 if 11 <= h < 17 else 0.06)
        if rng.random() < keen * evening * 0.045:
            length = rng.randint(25, 170) * 60
            out.append((t, min(t + length, END)))
            t += length + rng.randint(20, 120) * 60
        else:
            t += 300
    return out


SESSIONS = {name: sessions_for(name, keen) for name, (_, keen, _) in PLAYERS.items()}
SESSIONS[NEW_PLAYER] = [(NOW - 7 * 3600, NOW - 5 * 3600 - 600), (NOW - 3 * 3600, NOW + 3600)]
SESSIONS["frostbyte"].append((NOW - 50 * 60, END))
SESSIONS["MossyCobble"].append((NOW - 35 * 60, END))
SESSIONS["RedstoneRita"].append((NOW - 80 * 60, END))
RESTART = NOW - 9 * 3600 - 1200
DOWN = (RESTART - 60, RESTART + 30)
LAG = (NOW - 5 * 3600, NOW - 5 * 3600 + 1500)  # a hopper farm someone left running


def online(name, t):
    if DOWN[0] <= t < DOWN[1]:
        return False
    return any(a <= t < b for a, b in SESSIONS[name])


SESSIONS = {n: sorted(s) for n, s in SESSIONS.items()}

# ── Events, chat, and logs, drawn as the day goes.
CHAT = [
    "anyone up for the nether?", "found diamonds at y -58!!", "gg", "brb dinner", "who's got spare iron?",
    "a creeper got my wall again", "omw to the village", "nice build", "trading hall is done, come see",
    "lol", "need help with the warden?", "i'm at spawn", "thanks!", "where's the end portal again?",
    "sleep pls, phantoms", "selling mending books for 10 emeralds", "who left the hopper clock on",
    "back", "the bridge is finished", "good night all", "morning!", "anyone seen my wolf?",
    "raid at the village, come help", "just hit level 30", "this seed is wild", "ok that was close",
]
DEATHS = [
    ("minecraft:mob_attack", "minecraft:zombie", "was slain by Zombie", 5),
    ("minecraft:fall", "", "fell from a high place", 4),
    ("minecraft:arrow", "minecraft:skeleton", "was shot by Skeleton", 4),
    ("minecraft:explosion", "minecraft:creeper", "was blown up by Creeper", 3),
    ("minecraft:lava", "", "tried to swim in lava", 2),
    ("minecraft:mob_attack", "minecraft:spider", "was slain by Spider", 2),
    ("minecraft:drown", "", "drowned", 1),
    ("minecraft:mob_attack", "minecraft:piglin_brute", "was slain by Piglin Brute", 1),
    ("minecraft:sonic_boom", "minecraft:warden", "was obliterated by a sonically-charged shriek", 1),
    ("minecraft:player_attack", "minecraft:player", "was slain by frostbyte", 1),
]
ADVANCEMENTS = [
    ("Diamonds!", "task"), ("We Need to Go Deeper", "task"), ("Monster Hunter", "task"), ("Hot Stuff", "task"),
    ("Sweet Dreams", "task"), ("Isn't It Iron Pick", "task"), ("Cover Me with Diamonds", "task"),
    ("A Terrible Fortress", "task"), ("The End?", "task"), ("Hero of the Village", "challenge"),
    ("Sniper Duel", "challenge"), ("Bullseye", "challenge"), ("Into Fire", "task"), ("Total Beelocation", "goal"),
    ("Is It a Bird?", "task"), ("Return to Sender", "challenge"), ("Bring Home the Beacon", "goal"),
]
WARNINGS = [
    ("net.minecraft.server.network.ServerGamePacketListenerImpl", "warn", "{p} moved too quickly! 14.2,0.0,9.8", 6),
    ("net.minecraft.server.MinecraftServer", "warn", "Can't keep up! Is the server overloaded? Running 2140ms or 42 ticks behind", 0),
    ("CoreProtect", "warn", "Database busy, retrying the last write", 2),
    ("BlueMap", "warn", "Rendering the map 'world' takes longer than its update interval", 2),
    ("PlaceholderAPI", "error", "Failed to load expansion 'statistic': it needs a newer server", 1),
    ("TAB", "warn", "Placeholder %moafks_tag% took 52ms to compute", 1),
]

events, logs = [], []
event_id = 1000


def event(t, kind, player=None, data=None):
    global event_id
    event_id += 1
    item = {"id": event_id, "type": kind, "atMillis": t * 1000 + rng.randint(0, 999), "data": data or {}}
    if player:
        item["player"] = {"name": player, "uuid": f"00000000-0000-4000-8000-{abs(hash(player)) % 10**12:012d}"}
    events.append(item)


def log(t, logger, level, message):
    logs.append({"at": time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime(t)) + f".{rng.randint(0, 999):03d}Z",
                 "level": level, "logger": logger, "message": message})


# ── Lifetime statistics: a base before the window, then growth while online.
RATES = {  # per hour online
    "minecraft_player_deaths_total": 0.35, "minecraft_player_mob_kills_total": 42, "minecraft_player_player_kills_total": 0.03,
    "minecraft_player_damage_dealt_total": 640, "minecraft_player_damage_taken_total": 160, "minecraft_player_blocks_mined_total": 820,
    "minecraft_player_items_crafted_total": 130, "minecraft_player_distance_meters_total": 4300, "minecraft_player_jumps_total": 610,
    "minecraft_player_fish_caught_total": 2.5, "minecraft_player_animals_bred_total": 4, "minecraft_player_villager_trades_total": 9,
    "minecraft_player_items_enchanted_total": 1.6, "minecraft_player_diamonds_mined_total": 3.8,
    "minecraft_player_ancient_debris_mined_total": 0.4, "minecraft_player_experience_gained_total": 950,
    "minecraft_player_animals_tamed_total": 0.05,
}
stats = {}
for name, hours in [(n, h) for n, (h, _, _) in PLAYERS.items()] + list(DORMANT.items()):
    taste = {k: rng.uniform(0.5, 1.6) for k in RATES}
    stats[name] = {"taste": taste, "minecraft_player_play_time_seconds_total": hours * 3600,
                   "minecraft_player_advancements": min(120, int(hours * 0.28) + rng.randint(0, 6))}
    for k, rate in RATES.items():
        stats[name][k] = int(hours * rate * taste[k])
first_join = {n: NOW - int(h * 3600 * rng.uniform(4.5, 7)) for n, (h, _, _) in PLAYERS.items()}
first_join.update({n: NOW - int(h * 3600 * 9) for n, h in DORMANT.items()})
first_join[NEW_PLAYER] = SESSIONS[NEW_PLAYER][0][0]
last_seen = {n: NOW - rng.randint(2, 9) * 86400 for n in DORMANT}

counters = defaultdict(float)
for kind in ("player.join", "player.leave", "player.kick", "player.death", "player.advancement", "player.chat",
             "ext.quests.quest_completed", "ext.auctions.item_sold"):
    counters[("events", kind)] = 0
for frame in ("task", "goal", "challenge"):
    counters[("adv", frame)] = 0
heap = 2.0 * GiB
young_gc = 4810
young_gc_s = 61.2
old_gc = 3
conc_gc = 214
conc_gc_s = 9.1
world_time = 3000
day = 431
raining, thundering, weather_until = 0, 0, START + 3600
log_lines = defaultdict(float, {"info": 18000, "warn": 140, "error": 6})
problems = defaultdict(float)
deaths = defaultdict(float)
chat_by = defaultdict(float)
first_joins = 0.0
was_online = {n: False for n in PLAYERS}
where = {}
level = {n: rng.randint(8, 41) for n in PLAYERS}
pos = {n: [rng.uniform(-1800, 1800), 70, rng.uniform(-1800, 1800)] for n in PLAYERS}
disk_free = 142.6e9

for t in range(START, END + 1, STEP):
    down = DOWN[0] <= t < DOWN[1]
    lag = LAG[0] <= t < LAG[1]
    on = [n for n in PLAYERS if online(n, t)]
    n_on = len(on)

    for n in PLAYERS:
        now_on = n in on
        if now_on and not was_online[n]:
            counters[("events", "player.join")] += 1
            first = n == NEW_PLAYER and t == SESSIONS[n][0][0]
            if first:
                first_joins += 1
            event(t, "player.join", n, {"firstJoin": first})
            where[n] = rng.choices(["world", "world_nether", "world_the_end"], [0.78, 0.17, 0.05])[0]
        if was_online[n] and not now_on and not down:
            counters[("events", "player.leave")] += 1
            event(t, "player.leave", n)
            last_seen[n] = t
        was_online[n] = now_on
    if t == DOWN[0]:
        for n in PLAYERS:
            if was_online[n]:
                event(t, "player.leave", n)
    if down:
        continue
    for n in on:
        last_seen[n] = t

    # Server and performance.
    put(t, "minecraft_up", 1)
    put(t, "minecraft_info", 1, brand="Paper", minecraft_version="26.3", build="205", api_version="26.3.build.205-stable",
        worldapi_version="4.2.0", platform="paper", side="dedicated")
    put(t, "minecraft_paused", 0)
    put(t, "minecraft_start_time_seconds", RESTART if t >= RESTART else NOW - 4 * 86400 - 5000)
    put(t, "minecraft_players_online", n_on)
    put(t, "minecraft_players_max", 20)
    put(t, "minecraft_players_known", len(PLAYERS) + len(DORMANT) - (0 if t >= first_join[NEW_PLAYER] else 1))
    put(t, "minecraft_tick_rate", 20)
    put(t, "minecraft_whitelist_enabled", 1)
    put(t, "minecraft_datapacks_enabled", 4)

    p50 = 2.2 + 2.4 * n_on + rng.gauss(0, 0.4) + (26 if lag else 0)
    p50 = max(1.1, p50)
    p95, p99 = p50 * rng.uniform(1.6, 1.9), p50 * rng.uniform(2.2, 2.6)
    worst = p99 * rng.uniform(1.3, 1.8) + (rng.uniform(20, 45) if rng.random() < 0.04 else 0)
    put(t, "minecraft_tick_duration_seconds", p50 / 1000, quantile="0.5")
    put(t, "minecraft_tick_duration_seconds", p95 / 1000, quantile="0.95")
    put(t, "minecraft_tick_duration_seconds", p99 / 1000, quantile="0.99")
    put(t, "minecraft_tick_duration_seconds", worst / 1000, quantile="1")
    put(t, "minecraft_tick_duration_average_seconds", p50 * 1.12 / 1000)
    behind = max(0.0, (p95 - 50) / 50)
    tps1 = max(12.0, 20 - behind * 4.5 - (rng.uniform(0, 0.15) if n_on > 3 else 0))
    put(t, "minecraft_tps", round(tps1, 2), window="1m")
    prev5 = series[("minecraft_tps", (("window", "5m"),))]
    prev15 = series[("minecraft_tps", (("window", "15m"),))]
    put(t, "minecraft_tps", round((prev5[-1][1] * 9 + tps1) / 10 if prev5 else 20, 2), window="5m")
    put(t, "minecraft_tps", round((prev15[-1][1] * 29 + tps1) / 30 if prev15 else 20, 2), window="15m")

    cpu = min(0.95, 0.04 + 0.045 * n_on + (0.3 if lag else 0) + abs(rng.gauss(0, 0.015)))
    put(t, "minecraft_cpu_usage_ratio", cpu, scope="process")
    put(t, "minecraft_cpu_usage_ratio", min(0.98, cpu + 0.04 + abs(rng.gauss(0, 0.01))), scope="system")
    put(t, "minecraft_cpu_cores", 8)

    # The heap fills with play and drops at each young collection.
    heap += (60 + 55 * n_on + (400 if lag else 0)) * MiB * rng.uniform(0.7, 1.3)
    if heap > rng.uniform(4.6, 5.4) * GiB:
        heap = rng.uniform(1.6, 2.1) * GiB
        young_gc += 1
        young_gc_s += rng.uniform(0.012, 0.03)
    if rng.random() < 0.01:
        conc_gc += 1
        conc_gc_s += rng.uniform(0.01, 0.05)
    put(t, "minecraft_jvm_memory_used_bytes", heap, area="heap")
    put(t, "minecraft_jvm_memory_committed_bytes", 6 * GiB, area="heap")
    put(t, "minecraft_jvm_memory_max_bytes", 8 * GiB, area="heap")
    put(t, "minecraft_jvm_memory_used_bytes", 312 * MiB + rng.uniform(-4, 4) * MiB, area="nonheap")
    put(t, "minecraft_jvm_memory_committed_bytes", 340 * MiB, area="nonheap")
    old_gen = 1.15 * GiB + (t - START) / (END - START) * 0.3 * GiB
    for pool, area, used in [("G1 Eden Space", "heap", heap - old_gen - 40 * MiB), ("G1 Old Gen", "heap", old_gen),
                             ("G1 Survivor Space", "heap", 40 * MiB), ("Metaspace", "nonheap", 196 * MiB),
                             ("CodeHeap 'non-profiled nmethods'", "nonheap", 61 * MiB),
                             ("Compressed Class Space", "nonheap", 24 * MiB)]:
        put(t, "minecraft_jvm_memory_pool_used_bytes", max(used, 0), pool=pool, area=area)
    put(t, "minecraft_jvm_gc_collections_total", young_gc, gc="G1 Young Generation")
    put(t, "minecraft_jvm_gc_seconds_total", round(young_gc_s, 3), gc="G1 Young Generation")
    put(t, "minecraft_jvm_gc_collections_total", conc_gc, gc="G1 Concurrent GC")
    put(t, "minecraft_jvm_gc_seconds_total", round(conc_gc_s, 3), gc="G1 Concurrent GC")
    put(t, "minecraft_jvm_gc_collections_total", old_gc, gc="G1 Old Generation")
    put(t, "minecraft_jvm_gc_seconds_total", 0.84, gc="G1 Old Generation")
    put(t, "minecraft_jvm_threads", 64 + 2 * n_on, state="live")
    put(t, "minecraft_jvm_threads", 47 + n_on, state="daemon")
    put(t, "minecraft_jvm_threads", 91, state="peak")
    put(t, "minecraft_jvm_classes_loaded", 31877)
    put(t, "minecraft_jvm_buffer_pool_used_bytes", (38 + 3 * n_on) * MiB, pool="direct")
    put(t, "minecraft_jvm_buffer_pool_used_bytes", 0, pool="mapped")

    # The host.
    load = 0.35 + 0.42 * n_on + (2.8 if lag else 0)
    for window, smooth in (("1m", 1), ("5m", 5), ("15m", 15)):
        prev = series[("minecraft_host_load_average", (("window", window),))]
        put(t, "minecraft_host_load_average", round((prev[-1][1] * (smooth - 1) + load + rng.gauss(0, 0.08)) / smooth if prev else load, 2), window=window)
    put(t, "minecraft_host_cpus", 8)
    resident = 6 * GiB + 340 * MiB + 1.1 * GiB
    put(t, "minecraft_host_memory_total_bytes", 32 * GiB)
    put(t, "minecraft_host_memory_available_bytes", 32 * GiB - resident - 3.2 * GiB - rng.uniform(0, 0.2) * GiB)
    put(t, "minecraft_process_resident_memory_bytes", resident + rng.uniform(-30, 30) * MiB)
    disk_free -= rng.uniform(0, 2.4e5) * (1 + n_on)
    put(t, "minecraft_disk_total_bytes", 240e9)
    put(t, "minecraft_disk_free_bytes", disk_free)
    put(t, "minecraft_process_open_files", 410 + 9 * n_on + rng.randint(-3, 3))
    put(t, "minecraft_process_max_open_files", 1048576)
    put(t, "minecraft_java_info", 1, name="OpenJDK 64-Bit Server VM", vendor="Eclipse Adoptium", version="25.0.1")

    # Worlds.
    in_world = defaultdict(int)
    for n in on:
        in_world[where[n]] += 1
    world_time += 600
    if world_time >= 24000:
        world_time -= 24000
        day += 1
    if 12600 <= world_time < 23000 and in_world["world"] >= 2 and rng.random() < 0.18:
        world_time, day = 0, day + 1  # everyone slept
        event(t, "world.night_skipped", None, {"world": "world"})
        raining = thundering = 0
    if t >= weather_until:
        if raining:
            raining, thundering, weather_until = 0, 0, t + rng.randint(50, 140) * 60
        else:
            raining, weather_until = 1, t + rng.randint(8, 25) * 60
            thundering = 1 if rng.random() < 0.35 else 0
    for w, env, base, per in (("world", "normal", 520, 410), ("world_nether", "nether", 0, 330), ("world_the_end", "the_end", 0, 260)):
        chunks = base + per * in_world[w] + rng.randint(-12, 12) if (base or in_world[w]) else 0
        chunks = max(chunks, 0)
        ents = int(chunks * (1.05 if w == "world" else 0.7)) + (680 if lag and w == "world" else 0)
        put(t, "minecraft_world_info", 1, world=w, environment=env)
        put(t, "minecraft_world_chunks_loaded", chunks, world=w)
        put(t, "minecraft_world_entities", ents, world=w)
        put(t, "minecraft_world_block_entities", int(chunks * 0.62) + (90 if w == "world" else 0), world=w)
        put(t, "minecraft_world_players", in_world[w], world=w)
        put(t, "minecraft_world_days", day, world=w)
        put(t, "minecraft_world_time_ticks", world_time, world=w)
        put(t, "minecraft_world_raining", raining if w == "world" else 0, world=w)
        put(t, "minecraft_world_thundering", thundering if w == "world" else 0, world=w)
        put(t, "minecraft_world_border_size_blocks", 59999968, world=w)
        mix = {
            "world": [("item", 0.16 + (0.5 if lag else 0)), ("zombie", 0.12), ("cow", 0.1), ("skeleton", 0.09), ("sheep", 0.08),
                      ("villager", 0.07), ("creeper", 0.06), ("chicken", 0.06), ("spider", 0.05), ("bat", 0.04)],
            "world_nether": [("zombified_piglin", 0.38), ("piglin", 0.16), ("magma_cube", 0.12), ("ghast", 0.06),
                             ("strider", 0.08), ("hoglin", 0.07), ("item", 0.05)],
            "world_the_end": [("enderman", 0.82), ("shulker", 0.09), ("item", 0.03)],
        }[w]
        if ents:
            for kind, share in mix:
                put(t, "minecraft_world_entities_by_type", int(ents * share * rng.uniform(0.9, 1.1)), world=w, type=f"minecraft:{kind}")
        # What crowds the world.
        items = (34 + 6 * in_world[w] + rng.randint(-5, 5)) if w == "world" else 3 * in_world[w]
        if lag and w == "world":
            items = 640 + (t - LAG[0]) // 3
        put(t, "minecraft_world_dropped_items", items, world=w)
        put(t, "minecraft_world_dropped_item_count", int(items * 9.5), world=w)
        put(t, "minecraft_world_villagers", 64 if w == "world" else 0, world=w)
        put(t, "minecraft_world_tamed_mobs", (21 + (1 if t > NOW - 4 * 3600 else 0)) if w == "world" else 0, world=w)
        put(t, "minecraft_world_named_mobs", 9 if w == "world" else 0, world=w)
    for chunk, ents, tiles, hoppers in (("12,-8", 148, 62, 48), ("-3,21", 96, 18, 0), ("40,7", 71, 34, 12),
                                        ("0,0", 58, 41, 2), ("-17,-30", 44, 9, 0)):
        put(t, "minecraft_chunk_entities", ents + (520 if lag and chunk == "12,-8" else 0) + rng.randint(-4, 4), world="world", chunk=chunk)
        put(t, "minecraft_chunk_block_entities", tiles, world="world", chunk=chunk)
        put(t, "minecraft_chunk_hoppers", hoppers, world="world", chunk=chunk)

    # Players online.
    for n in on:
        s = stats[n]
        s["minecraft_player_play_time_seconds_total"] += STEP
        for k, rate in RATES.items():
            s[k] += rate * s["taste"][k] * STEP / 3600 * rng.uniform(0, 2)
        pos[n][0] += rng.uniform(-40, 40)
        pos[n][2] += rng.uniform(-40, 40)
        pos[n][1] = min(max(pos[n][1] + rng.uniform(-6, 6), -58), 140)
        if rng.random() < 0.004:
            level[n] += 1
        ping = PLAYERS[n][2] * rng.uniform(0.85, 1.25)
        health = max(1.0, min(20.0, 20 - abs(rng.gauss(0, 5))))
        put(t, "minecraft_player_info", 1, player=n, uuid=f"uuid-{n}", world=where[n], gamemode="survival")
        put(t, "minecraft_player_ping_seconds", round(ping) / 1000, player=n)
        put(t, "minecraft_player_health", round(health * 2) / 2, player=n)
        put(t, "minecraft_player_food", max(3, min(20, int(20 - abs(rng.gauss(0, 5))))), player=n)
        put(t, "minecraft_player_level", level[n], player=n)
        put(t, "minecraft_player_idle_seconds", rng.choice([0, 0, 0, 4, 12, 35, 140]), player=n)
        for axis, i in (("x", 0), ("y", 1), ("z", 2)):
            put(t, "minecraft_player_position_blocks", round(pos[n][i], 1), player=n, world=where[n], axis=axis)
        # What they do.
        if rng.random() < 0.012:
            cause, killer, msg, _ = rng.choices(DEATHS, [d[3] for d in DEATHS])[0]
            deaths[(cause, killer)] += 1
            counters[("events", "player.death")] += 1
            s["minecraft_player_deaths_total"] += 1
            event(t, "player.death", n, {"cause": cause, "message": f"{n} {msg}", "killer": {"type": killer} if killer else None})
        if rng.random() < 0.01:
            title, frame = rng.choice(ADVANCEMENTS)
            counters[("adv", frame)] += 1
            counters[("events", "player.advancement")] += 1
            s["minecraft_player_advancements"] += 1
            event(t, "player.advancement", n, {"title": title, "frame": frame})
        if rng.random() < 0.05 * (1 + n_on / 4):
            counters[("events", "player.chat")] += 1
            chat_by[n] += 1
            event(t, "player.chat", n, {"message": rng.choice(CHAT)})
        if rng.random() < 0.006:
            counters[("events", "ext.quests.quest_completed")] += 1
        if rng.random() < 0.004:
            counters[("events", "ext.auctions.item_sold")] += 1

    for n, s in stats.items():
        if n == NEW_PLAYER and t < first_join[n]:
            continue
        for k, v in s.items():
            if k == "taste":
                continue
            put(t, k, int(v) if k != "minecraft_player_play_time_seconds_total" else v, player=n)
        if t >= first_join[n]:
            put(t, "minecraft_player_first_join_timestamp_seconds", first_join[n], player=n)
        if n in last_seen:
            put(t, "minecraft_player_last_seen_timestamp_seconds", last_seen[n], player=n)

    # The console.
    log_lines["info"] += rng.randint(2, 6) + 3 * n_on
    for logger, lvl, msg, weight in WARNINGS:
        chance = weight * 0.0016 * (1 + n_on) + (0.25 if lag and "overloaded" in msg else 0)
        if rng.random() < chance:
            problems[(lvl, logger)] += 1
            log_lines[lvl] += 1
            log(t, logger, lvl, msg.format(p=rng.choice(on) if on else "frostbyte"))

    for (kind, key), v in counters.items():
        if kind == "events":
            put(t, "minecraft_events_total", v, type=key)
        else:
            put(t, "minecraft_advancements_total", v, frame=key)
    for (cause, killer), v in deaths.items():
        put(t, "minecraft_deaths_total", v, cause=cause, killer=killer)
    for n, v in chat_by.items():
        put(t, "minecraft_player_chat_messages_total", v, player=n)
    put(t, "minecraft_first_joins_total", first_joins)
    for lvl, v in log_lines.items():
        put(t, "minecraft_log_messages_total", v, level=lvl)
    for (lvl, logger), v in problems.items():
        put(t, "minecraft_log_problems_total", v, level=lvl, logger=logger)

    for plugin, version in (("WorldAPI", "3.2.0"), ("LuckPerms", "5.5.17"), ("CoreProtect", "23.1"), ("GriefPrevention", "16.18.7"),
                            ("BlueMap", "5.29"), ("Chunky", "1.5.3"), ("PlaceholderAPI", "2.11.7"), ("TAB", "6.1.2"),
                            ("ViaVersion", "5.6.0"), ("Mo'Joins", "2.3.1"), ("Mo'Afks", "2.2.1")):
        put(t, "minecraft_plugin_enabled", 1, plugin=plugin.replace("'", "’"), version=version, kind="plugin")
    put(t, "minecraft_exporter_scrape_duration_seconds", round(rng.uniform(0.018, 0.06) + 0.004 * n_on, 4))
    for endpoint in ("server", "server/performance", "worlds", "players/online", "players", "leaderboards", "events",
                     "extensions", "server/system", "server/logs", "server/plugins", "worlds/entities/summary"):
        put(t, "minecraft_exporter_endpoint_allowed", 1, endpoint=endpoint)

for endpoint in ("server", "server/performance", "worlds", "players/online"):
    for t in range(DOWN[1], END + 1, STEP):
        put(t, "minecraft_exporter_request_errors_total", 2, endpoint=endpoint, target="default")

with open(os.path.join(OUT, "metrics.om"), "w") as f:
    by_name = defaultdict(list)
    for (name, labels), samples in series.items():
        by_name[name].append((labels, samples))
    for name, items in by_name.items():
        for labels, samples in items:
            text = ",".join(f'{k}="{v}"' for k, v in sorted({**BASE, **dict(labels)}.items()))
            for t, v in samples:
                f.write(f"{name}{{{text}}} {v:.6g} {t}\n" if isinstance(v, float) else f"{name}{{{text}}} {v} {t}\n")
    f.write("# EOF\n")

events.sort(key=lambda e: e["atMillis"])
json.dump(events, open(os.path.join(OUT, "events.json"), "w"))
logs.sort(key=lambda l: l["at"])
json.dump(logs, open(os.path.join(OUT, "logs.json"), "w"))
print(f"{len(series)} series, {sum(len(s) for s in series.values())} samples, {len(events)} events, {len(logs)} log lines")
