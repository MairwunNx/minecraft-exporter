package collector

import (
	"strconv"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
)

func collectServer(out *emitter, s *worldapi.Server) {
	build := ""
	if s.Build != nil {
		build = strconv.Itoa(*s.Build)
	}
	out.gauge("minecraft_info", "The server's software and versions", 1,
		"brand", s.Name, "minecraft_version", s.MinecraftVersion, "build", build,
		"api_version", s.APIVersion, "worldapi_version", s.WorldAPIVersion)
	if started, err := time.Parse(time.RFC3339Nano, s.StartedAt); err == nil {
		out.gauge("minecraft_start_time_seconds", "When the server started, in Unix seconds", float64(started.UnixMilli())/1000)
	}
	out.gauge("minecraft_players_online", "Players online now", float64(s.OnlinePlayers))
	out.gauge("minecraft_players_max", "The player limit", float64(s.MaxPlayers))
	out.gauge("minecraft_tick_rate", "Ticks per second the server aims for", s.TickRate)
	out.gauge("minecraft_tick_frozen", "1 while the game's ticks are frozen", boolValue(s.Frozen))
	out.gauge("minecraft_whitelist_enabled", "1 when only whitelisted players may join", boolValue(s.Whitelist))
	enabled := 0
	for _, pack := range s.Datapacks {
		if pack.Enabled {
			enabled++
		}
	}
	out.gauge("minecraft_datapacks_enabled", "Datapacks enabled, built-in ones included", float64(enabled))
}

func collectPerformance(out *emitter, p *worldapi.Performance) {
	const tpsHelp = "Ticks per second, averaged over the window"
	out.gauge("minecraft_tps", tpsHelp, p.TPS.OneMinute, "window", "1m")
	out.gauge("minecraft_tps", tpsHelp, p.TPS.FiveMinutes, "window", "5m")
	out.gauge("minecraft_tps", tpsHelp, p.TPS.FifteenMinutes, "window", "15m")

	const tickHelp = "How long a tick takes, by quantile over recent ticks; 1 is the slowest"
	out.gauge("minecraft_tick_duration_seconds", tickHelp, p.MSPT.P50/1000, "quantile", "0.5")
	out.gauge("minecraft_tick_duration_seconds", tickHelp, p.MSPT.P95/1000, "quantile", "0.95")
	out.gauge("minecraft_tick_duration_seconds", tickHelp, p.MSPT.P99/1000, "quantile", "0.99")
	out.gauge("minecraft_tick_duration_seconds", tickHelp, p.MSPT.Max/1000, "quantile", "1")
	out.gauge("minecraft_tick_duration_average_seconds", "How long a tick takes on average over recent ticks", p.MSPT.Average/1000)

	const cpuHelp = "CPU in use, from 0 to 1 across every core: the server's process or the whole system"
	if p.CPU.Process != nil {
		out.gauge("minecraft_cpu_usage_ratio", cpuHelp, *p.CPU.Process, "scope", "process")
	}
	if p.CPU.System != nil {
		out.gauge("minecraft_cpu_usage_ratio", cpuHelp, *p.CPU.System, "scope", "system")
	}
	out.gauge("minecraft_cpu_cores", "CPU cores the server sees", float64(p.CPU.Cores))

	for _, w := range p.Worlds {
		out.gauge("minecraft_world_chunks_loaded", "Chunks loaded in the world", float64(w.LoadedChunks), "world", w.World)
		out.gauge("minecraft_world_entities", "Entities in the world's loaded chunks", float64(w.Entities), "world", w.World)
		out.gauge("minecraft_world_block_entities", "Block entities, like chests and hoppers, in the world's loaded chunks", float64(w.TileEntities), "world", w.World)
		out.gauge("minecraft_world_players", "Players in the world", float64(w.Players), "world", w.World)
		for kind, count := range w.EntityTypes {
			out.gauge("minecraft_world_entities_by_type", "Entities of the world's most common types", float64(count), "world", w.World, "type", kind)
		}
	}

	if p.JVM == nil {
		return
	}
	jvm := p.JVM
	memory(out, "heap", jvm.Heap)
	memory(out, "nonheap", jvm.NonHeap)
	for _, pool := range jvm.Pools {
		area := "heap"
		if pool.Type == "non_heap" {
			area = "nonheap"
		}
		out.gauge("minecraft_jvm_memory_pool_used_bytes", "Memory a pool uses", float64(pool.UsedBytes), "pool", pool.Name, "area", area)
		out.gauge("minecraft_jvm_memory_pool_committed_bytes", "Memory the JVM holds for a pool", float64(pool.CommittedBytes), "pool", pool.Name, "area", area)
		if pool.MaxBytes != nil {
			out.gauge("minecraft_jvm_memory_pool_max_bytes", "The most memory a pool may use", float64(*pool.MaxBytes), "pool", pool.Name, "area", area)
		}
	}
	for _, gc := range jvm.GC {
		out.counter("minecraft_jvm_gc_collections_total", "Garbage collections by collector", float64(gc.Count), "gc", gc.Name)
		out.counter("minecraft_jvm_gc_seconds_total", "Time spent collecting garbage, by collector", float64(gc.TimeMillis)/1000, "gc", gc.Name)
	}
	const threadsHelp = "JVM threads: live now, daemons among them, and the peak since start"
	out.gauge("minecraft_jvm_threads", threadsHelp, float64(jvm.Threads.Live), "state", "live")
	out.gauge("minecraft_jvm_threads", threadsHelp, float64(jvm.Threads.Daemon), "state", "daemon")
	out.gauge("minecraft_jvm_threads", threadsHelp, float64(jvm.Threads.Peak), "state", "peak")
	out.gauge("minecraft_jvm_classes_loaded", "Classes loaded now", float64(jvm.ClassesLoaded))
	for _, buffer := range jvm.Buffers {
		out.gauge("minecraft_jvm_buffer_pool_used_bytes", "Memory a buffer pool uses", float64(buffer.UsedBytes), "pool", buffer.Name)
		out.gauge("minecraft_jvm_buffer_pool_capacity_bytes", "Capacity of a buffer pool's buffers", float64(buffer.CapacityBytes), "pool", buffer.Name)
		out.gauge("minecraft_jvm_buffer_pool_buffers", "Buffers in a pool", float64(buffer.Count), "pool", buffer.Name)
	}
}

func memory(out *emitter, area string, m worldapi.Memory) {
	out.gauge("minecraft_jvm_memory_used_bytes", "JVM memory in use", float64(m.UsedBytes), "area", area)
	out.gauge("minecraft_jvm_memory_committed_bytes", "JVM memory the system has given the JVM", float64(m.CommittedBytes), "area", area)
	if m.MaxBytes != nil {
		out.gauge("minecraft_jvm_memory_max_bytes", "The most memory the JVM may use", float64(*m.MaxBytes), "area", area)
	}
}

func collectWorlds(out *emitter, worlds []worldapi.World) {
	for _, w := range worlds {
		out.gauge("minecraft_world_info", "A world and its environment: normal, nether, or the_end", 1, "world", w.Name, "environment", w.Environment)
		out.counter("minecraft_world_days_total", "Days the world has lived", float64(w.Time.Day), "world", w.Name)
		out.gauge("minecraft_world_time_ticks", "Time of day, from 0 at sunrise to 23999", float64(w.Time.DayTime), "world", w.Name)
		out.gauge("minecraft_world_raining", "1 while it rains or snows", boolValue(w.Weather.Raining), "world", w.Name)
		out.gauge("minecraft_world_thundering", "1 during a thunderstorm", boolValue(w.Weather.Thundering), "world", w.Name)
		out.gauge("minecraft_world_border_size_blocks", "The world border's side", w.Border.Size, "world", w.Name)
	}
}

func collectOnline(out *emitter, online *worldapi.OnlinePlayers, perPlayer bool) {
	if !perPlayer {
		return
	}
	for _, p := range online.Players {
		out.gauge("minecraft_player_info", "A player online now: where they are and their game mode", 1,
			"player", p.Name, "uuid", p.UUID, "world", p.World, "gamemode", p.GameMode)
		out.gauge("minecraft_player_ping_seconds", "A player's ping", float64(p.Ping)/1000, "player", p.Name)
		out.gauge("minecraft_player_health", "A player's health, 20 when full", p.Health, "player", p.Name)
		out.gauge("minecraft_player_food", "A player's hunger bar, 20 when full", float64(p.Food), "player", p.Name)
		out.gauge("minecraft_player_level", "A player's experience level", float64(p.Level), "player", p.Name)
		out.gauge("minecraft_player_idle_seconds", "How long a player hasn't moved or acted", float64(p.IdleSeconds), "player", p.Name)
		if l := p.Location; l != nil {
			const help = "Where a player stands, by axis; only for clients that may see coordinates"
			out.gauge("minecraft_player_position_blocks", help, l.X, "player", p.Name, "world", l.World, "axis", "x")
			out.gauge("minecraft_player_position_blocks", help, l.Y, "player", p.Name, "world", l.World, "axis", "y")
			out.gauge("minecraft_player_position_blocks", help, l.Z, "player", p.Name, "world", l.World, "axis", "z")
		}
	}
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
