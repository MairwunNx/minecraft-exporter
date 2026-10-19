package collector

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
)

// Data that needs a token: WorldAPI keeps the host, the console, plugins, moderation, and what crowds
// each world from anonymous clients. Without the permission, the endpoint stays closed and its metrics away.

// topChunks is how many of each world's busiest chunks the exporter names.
const topChunks = 5

// logLines is how many of the newest console lines one scrape reads.
const logLines = 1000

func (t *Target) privateItems() []slowItem {
	items := []slowItem{
		{"plugins", "server/plugins", func(ctx context.Context) (any, error) { return t.client.GetPlugins(ctx) }},
		{"moderation", "server/bans", func(ctx context.Context) (any, error) { return t.client.GetModeration(ctx) }},
	}
	t.accessMu.Lock()
	worlds, bosses := slices.Clone(t.worlds), t.hasExtension("mobosses")
	t.accessMu.Unlock()
	if bosses {
		items = append(items, slowItem{"mobosses", "extensions/mobosses/stats", func(ctx context.Context) (any, error) { return t.client.GetBossStats(ctx) }})
	}
	for _, world := range worlds {
		items = append(items, slowItem{"summary:" + world, "worlds/entities/summary", func(ctx context.Context) (any, error) {
			return t.client.GetEntitySummary(ctx, world)
		}})
	}
	return items
}

// hasExtension says whether a plugin shares the namespace through WorldAPI; call it under accessMu.
func (t *Target) hasExtension(namespace string) bool {
	return slices.Contains(t.namespaces, namespace)
}

func (t *Target) collectPrivate(out *emitter) {
	t.slowMu.Lock()
	defer t.slowMu.Unlock()

	if plugins, ok := t.slow.values["plugins"].([]worldapi.Plugin); ok {
		for _, p := range plugins {
			out.gauge("minecraft_plugin_enabled", "A plugin on the server and its version: 1 while enabled", boolValue(p.Enabled),
				"plugin", p.Name, "version", p.Version)
		}
	}
	if m, ok := t.slow.values["moderation"].(*worldapi.Moderation); ok {
		out.gauge("minecraft_bans", "Bans in force, of players or of IP addresses", float64(m.BannedPlayers), "kind", "player")
		out.gauge("minecraft_bans", "Bans in force, of players or of IP addresses", float64(m.BannedAddresses), "kind", "ip")
		out.gauge("minecraft_operators", "Operators", float64(m.Operators))
		out.gauge("minecraft_whitelist_players", "Players on the whitelist", float64(m.Whitelisted))
	}
	if b, ok := t.slow.values["mobosses"].(*worldapi.BossStats); ok {
		out.gauge("minecraft_mobosses_alive", "Mo’Bosses bosses alive now", float64(b.Alive))
		for _, c := range b.Counts {
			out.counter("minecraft_mobosses_spawned_total", "Mo’Bosses bosses spawned, by type and class", float64(c.Spawned), "type", c.Type, "class", c.Class)
			out.counter("minecraft_mobosses_killed_total", "Mo’Bosses bosses killed, by type and class", float64(c.Killed), "type", c.Type, "class", c.Class)
		}
	}
	for key, value := range t.slow.values {
		world, ok := strings.CutPrefix(key, "summary:")
		summary, _ := value.(*worldapi.EntitySummary)
		if !ok || summary == nil {
			continue
		}
		out.gauge("minecraft_world_dropped_items", "Item stacks lying on the ground in loaded chunks", float64(summary.DroppedItems), "world", world)
		out.gauge("minecraft_world_dropped_item_count", "Items lying on the ground in loaded chunks, stacks added up", float64(summary.DroppedItemCount), "world", world)
		out.gauge("minecraft_world_villagers", "Villagers in loaded chunks", float64(summary.Villagers), "world", world)
		out.gauge("minecraft_world_named_mobs", "Mobs with a name tag in loaded chunks", float64(summary.Named), "world", world)
		out.gauge("minecraft_world_tamed_mobs", "Tamed animals in loaded chunks", float64(summary.Tamed), "world", world)
		for _, c := range summary.TopChunks[:min(len(summary.TopChunks), topChunks)] {
			chunk := fmt.Sprintf("%d,%d", c.X, c.Z)
			out.gauge("minecraft_chunk_entities", "Entities in the world's busiest chunks, by chunk coordinates", float64(c.Entities), "world", world, "chunk", chunk)
			out.gauge("minecraft_chunk_block_entities", "Block entities in the world's busiest chunks", float64(c.TileEntities), "world", world, "chunk", chunk)
			out.gauge("minecraft_chunk_hoppers", "Hoppers in the world's busiest chunks", float64(c.Hoppers), "world", world, "chunk", chunk)
		}
	}
}

func collectSystem(out *emitter, s *worldapi.System) {
	for i, window := range []string{"1m", "5m", "15m"} {
		if i < len(s.LoadAverage) {
			out.gauge("minecraft_host_load_average", "The host's load average over the window", s.LoadAverage[i], "window", window)
		}
	}
	out.gauge("minecraft_host_cpus", "CPUs the host has", float64(s.CPUs))
	optional := func(name, help string, value *int64) {
		if value != nil {
			out.gauge(name, help, float64(*value))
		}
	}
	optional("minecraft_host_memory_total_bytes", "The host's memory", s.Memory.TotalBytes)
	optional("minecraft_host_memory_available_bytes", "Memory the host can still give out", s.Memory.AvailableBytes)
	optional("minecraft_host_swap_total_bytes", "The host's swap", s.Memory.SwapTotalBytes)
	optional("minecraft_host_swap_free_bytes", "Swap not in use", s.Memory.SwapFreeBytes)
	out.gauge("minecraft_disk_total_bytes", "The disk under the server folder", float64(s.Disk.TotalBytes))
	out.gauge("minecraft_disk_free_bytes", "Space the server may still use on its disk", float64(s.Disk.UsableBytes))
	if c := s.Container; c != nil {
		optional("minecraft_container_memory_limit_bytes", "The container's memory limit", c.MemoryLimitBytes)
		optional("minecraft_container_memory_used_bytes", "Memory the container uses", c.MemoryUsedBytes)
		if c.CPULimit != nil {
			out.gauge("minecraft_container_cpu_limit", "CPUs the container may use", *c.CPULimit)
		}
	}
	optional("minecraft_process_open_files", "Files the server has open", s.OpenFiles)
	optional("minecraft_process_max_open_files", "The most files the server may open", s.MaxOpenFiles)
	optional("minecraft_process_resident_memory_bytes", "Memory the server's process holds, the JVM's own included", s.ProcessResidentBytes)
	out.gauge("minecraft_java_info", "The Java runtime the server runs on", 1, "name", s.Java.Name, "vendor", s.Java.Vendor, "version", s.Java.Version)
}

// logCounts count console lines since the exporter started; the first poll only marks where to start.
type logCounts struct {
	started bool
	lastAt  time.Time
	seen    map[string]bool // lines at lastAt already counted: several share a millisecond
	byLevel map[string]float64
	trouble map[[2]string]float64 // level, logger of warnings and errors
}

func newLogCounts() logCounts {
	counts := logCounts{seen: map[string]bool{}, byLevel: map[string]float64{}, trouble: map[[2]string]float64{}}
	for _, level := range []string{"info", "warn", "error"} {
		counts.byLevel[level] = 0
	}
	return counts
}

func (t *Target) pollLogs(ctx context.Context) {
	if !t.allowed("server/logs") {
		return
	}
	lines, err := t.client.GetLogs(ctx, logLines)
	if !t.done("server/logs", err) {
		return
	}
	t.logsMu.Lock()
	defer t.logsMu.Unlock()
	l := &t.logs
	for _, line := range lines {
		at, err := time.Parse(time.RFC3339Nano, line.At)
		if err != nil || at.Before(l.lastAt) {
			continue
		}
		key := line.Level + "\x00" + line.Message
		if at.After(l.lastAt) {
			l.lastAt, l.seen = at, map[string]bool{}
		} else if l.seen[key] {
			continue
		}
		l.seen[key] = true
		if !l.started {
			continue
		}
		l.byLevel[line.Level]++
		if line.Level == "warn" || line.Level == "error" || line.Level == "fatal" {
			logger := ""
			if line.Logger != nil {
				logger = *line.Logger
			}
			l.trouble[[2]string{line.Level, logger}]++
		}
	}
	l.started = true
}

func (t *Target) collectLogs(out *emitter) {
	t.logsMu.Lock()
	defer t.logsMu.Unlock()
	if !t.logs.started {
		return
	}
	for level, n := range t.logs.byLevel {
		out.counter("minecraft_log_messages_total", "Console lines since the exporter started, by level", n, "level", level)
	}
	for key, n := range t.logs.trouble {
		out.counter("minecraft_log_problems_total", "Warnings and errors in the console since the exporter started, by who logged them", n,
			"level", key[0], "logger", key[1])
	}
}
