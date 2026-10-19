package collector

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
)

// statistic is one leaderboard turned into a per-player metric.
type statistic struct {
	board  string  // a WorldAPI preset or a vanilla statistic
	metric string  // the metric's name
	help   string  // its description
	scale  float64 // turns the board's unit into the metric's
	gauge  bool    // a count that may go down, unlike the statistics that only grow
}

// statistics are the leaderboards the exporter reads. Vanilla statistics come raw: play time in
// ticks, damage in tenths of a health point. Presets cover what vanilla doesn't count.
var statistics = []statistic{
	{"custom:play_time", "minecraft_player_play_time_seconds_total", "Time played", 1.0 / 20, false},
	{"custom:deaths", "minecraft_player_deaths_total", "Deaths", 1, false},
	{"custom:mob_kills", "minecraft_player_mob_kills_total", "Mobs killed", 1, false},
	{"custom:player_kills", "minecraft_player_player_kills_total", "Players killed", 1, false},
	{"custom:damage_dealt", "minecraft_player_damage_dealt_total", "Damage dealt, in health points (2 per heart)", 0.1, false},
	{"custom:damage_taken", "minecraft_player_damage_taken_total", "Damage taken, in health points (2 per heart)", 0.1, false},
	{"mined:*", "minecraft_player_blocks_mined_total", "Blocks mined", 1, false},
	{"crafted:*", "minecraft_player_items_crafted_total", "Items crafted", 1, false},
	{"travel", "minecraft_player_distance_meters_total", "Distance traveled in every way", 1000, false},
	{"custom:jump", "minecraft_player_jumps_total", "Jumps", 1, false},
	{"custom:fish_caught", "minecraft_player_fish_caught_total", "Fish caught", 1, false},
	{"custom:animals_bred", "minecraft_player_animals_bred_total", "Animals bred", 1, false},
	{"custom:traded_with_villager", "minecraft_player_villager_trades_total", "Trades with villagers", 1, false},
	{"custom:enchant_item", "minecraft_player_items_enchanted_total", "Items enchanted", 1, false},
	{"diamonds", "minecraft_player_diamonds_mined_total", "Diamond ores mined", 1, false},
	{"netherite", "minecraft_player_ancient_debris_mined_total", "Ancient debris mined", 1, false},
	{"experience", "minecraft_player_experience_gained_total", "Experience points gained, as WorldAPI counts them", 1, false},
	{"tamed", "minecraft_player_animals_tamed_total", "Animals tamed, as WorldAPI counts them", 1, false},
	{"advancements", "minecraft_player_advancements", "Advancements finished, recipes left out", 1, true},
}

// statsBatch is how many statistics one scrape refreshes at most. WorldAPI lets an anonymous client
// burst 30 requests, so the cache refreshes a few boards per scrape instead of all at once.
const statsBatch = 6

// playersKey stands for the player list among the boards in the cache.
const playersKey = "players"

type statsCache struct {
	read    map[string]time.Time // when each board, and the player list, was last tried
	boards  map[string]*worldapi.Leaderboard
	players []worldapi.PlayerSummary
}

// refreshStats reads again the stalest statistics that are older than the interval, a batch at a time.
// A board that fails keeps its last answer.
func (t *Target) refreshStats(ctx context.Context) {
	t.statsMu.Lock()
	defer t.statsMu.Unlock()
	if t.stats.read == nil {
		t.stats.read, t.stats.boards = map[string]time.Time{}, map[string]*worldapi.Leaderboard{}
	}

	due := []string{}
	for _, key := range append([]string{playersKey}, boardNames()...) {
		if time.Since(t.stats.read[key]) >= t.opts.StatsInterval {
			due = append(due, key)
		}
	}
	slices.SortStableFunc(due, func(a, b string) int { return t.stats.read[a].Compare(t.stats.read[b]) })
	due = due[:min(len(due), statsBatch)]

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, key := range due {
		t.stats.read[key] = time.Now()
		wg.Go(func() {
			if key == playersKey {
				players, err := t.client.GetPlayers(ctx)
				t.failed("players", err)
				if err == nil {
					mu.Lock()
					t.stats.players = players
					mu.Unlock()
				}
				return
			}
			board, err := t.client.GetLeaderboard(ctx, key)
			t.failed("leaderboards", err)
			if err == nil {
				mu.Lock()
				t.stats.boards[key] = board
				mu.Unlock()
			}
		})
	}
	wg.Wait()
}

func boardNames() []string {
	names := make([]string, len(statistics))
	for i, s := range statistics {
		names[i] = s.board
	}
	return names
}

func (t *Target) collectStats(out *emitter) {
	t.statsMu.Lock()
	defer t.statsMu.Unlock()

	for _, s := range statistics {
		board := t.stats.boards[s.board]
		if board == nil {
			continue
		}
		emit := out.counter
		if s.gauge {
			emit = out.gauge
		}
		if !t.opts.Players {
			sum := 0.0
			for _, entry := range board.Entries {
				sum += entry.Value
			}
			emit(s.metric, s.help+", every player added up", sum*s.scale)
			continue
		}
		for _, entry := range board.Entries {
			emit(s.metric, s.help, entry.Value*s.scale, "player", entry.Player.Name)
		}
	}

	if t.stats.players == nil {
		return
	}
	out.gauge("minecraft_players_known", "Players who have ever joined", float64(len(t.stats.players)))
	if !t.opts.Players {
		return
	}
	for _, p := range t.stats.players {
		if at, ok := parseTime(p.FirstPlayed); ok {
			out.gauge("minecraft_player_first_join_timestamp_seconds", "When a player first joined, in Unix seconds", at, "player", p.Name)
		}
		if at, ok := parseTime(p.LastSeen); ok {
			out.gauge("minecraft_player_last_seen_timestamp_seconds", "When a player was last online, in Unix seconds", at, "player", p.Name)
		}
	}
}

func parseTime(value *string) (float64, bool) {
	if value == nil {
		return 0, false
	}
	at, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return 0, false
	}
	return float64(at.UnixMilli()) / 1000, true
}
