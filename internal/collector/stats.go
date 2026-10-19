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

// slowBatch is how many slow items one scrape refreshes at most. WorldAPI lets an anonymous client
// burst 30 requests, so the cache refreshes a few items per scrape instead of all at once; a token
// allows more.
const (
	slowBatch      = 6
	slowBatchToken = 12
)

// slowItem is data read on the slow cadence: a leaderboard, the player list, or token-only data.
type slowItem struct {
	key      string
	endpoint string // names it in errors and in minecraft_exporter_endpoint_allowed
	fetch    func(context.Context) (any, error)
}

type slowCache struct {
	read   map[string]time.Time // when each item was last tried
	values map[string]any
}

const playersKey = "players"

func boardKey(board string) string { return "board:" + board }

// slowItems lists everything read on the slow cadence: statistics for everyone, then what a token unlocks.
func (t *Target) slowItems() []slowItem {
	items := []slowItem{{playersKey, "players", func(ctx context.Context) (any, error) { return t.client.GetPlayers(ctx) }}}
	for _, s := range statistics {
		items = append(items, slowItem{boardKey(s.board), "leaderboards", func(ctx context.Context) (any, error) {
			return t.client.GetLeaderboard(ctx, s.board)
		}})
	}
	return append(items, t.privateItems()...)
}

// refreshSlow reads again the stalest items that are older than the interval, a batch at a time.
// An item that fails keeps its last answer; one the token may not read waits until it's allowed.
func (t *Target) refreshSlow(ctx context.Context) {
	t.slowMu.Lock()
	defer t.slowMu.Unlock()
	if t.slow.read == nil {
		t.slow.read, t.slow.values = map[string]time.Time{}, map[string]any{}
	}

	due := []slowItem{}
	for _, item := range t.slowItems() {
		if time.Since(t.slow.read[item.key]) >= t.opts.StatsInterval && t.allowed(item.endpoint) {
			due = append(due, item)
		}
	}
	slices.SortStableFunc(due, func(a, b slowItem) int { return t.slow.read[a.key].Compare(t.slow.read[b.key]) })
	batch := slowBatch
	if t.opts.Token {
		batch = slowBatchToken
	}
	due = due[:min(len(due), batch)]

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, item := range due {
		t.slow.read[item.key] = time.Now()
		wg.Go(func() {
			value, err := item.fetch(ctx)
			if t.done(item.endpoint, err) {
				mu.Lock()
				t.slow.values[item.key] = value
				mu.Unlock()
			}
		})
	}
	wg.Wait()
}

func (t *Target) collectStats(out *emitter) {
	t.slowMu.Lock()
	defer t.slowMu.Unlock()

	for _, s := range statistics {
		board, _ := t.slow.values[boardKey(s.board)].(*worldapi.Leaderboard)
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

	players, ok := t.slow.values[playersKey].([]worldapi.PlayerSummary)
	if !ok {
		return
	}
	out.gauge("minecraft_players_known", "Players who have ever joined", float64(len(players)))
	if !t.opts.Players {
		return
	}
	for _, p := range players {
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
