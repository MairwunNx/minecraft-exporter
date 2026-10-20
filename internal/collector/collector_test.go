package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// fakeWorldAPI answers like WorldAPI with two players, Steve online and Alex offline.
type fakeWorldAPI struct {
	mu        sync.Mutex
	events    []map[string]any
	logs      []map[string]any
	forbidden map[string]bool
	requests  map[string]int
}

func (f *fakeWorldAPI) addEvent(id int64, kind string, data map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, map[string]any{"id": id, "type": kind, "atMillis": 1_700_000_000_000 + id, "data": data,
		"player": map[string]any{"uuid": "u-steve", "name": "Steve"}})
}

func (f *fakeWorldAPI) addLog(at, level, logger, message string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, map[string]any{"at": at, "level": level, "logger": logger, "message": message})
}

func (f *fakeWorldAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	f.requests[path]++
	for prefix := range f.forbidden {
		if strings.HasPrefix(path, prefix) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"forbidden","message":"no"}}`))
			return
		}
	}
	var body any
	switch {
	case path == "/server":
		body = map[string]any{"name": "Paper", "minecraftVersion": "26.3", "build": 205, "apiVersion": "26.3.build.205-stable",
			"worldApiVersion": "3.2.0", "maxPlayers": 20, "onlinePlayers": 1, "startedAt": "2026-10-19T15:11:13.423Z", "tickRate": 20.0,
			"datapacks": []map[string]any{{"enabled": true}, {"enabled": false}}}
	case path == "/server/performance":
		body = map[string]any{
			"tps":  map[string]any{"oneMinute": 19.5, "fiveMinutes": 19.9, "fifteenMinutes": 20.0},
			"mspt": map[string]any{"average": 9.6, "p50": 9.3, "p95": 12.5, "p99": 14.0, "max": 22.0},
			"cpu":  map[string]any{"process": 0.5, "system": 0.6, "cores": 8},
			"worlds": []map[string]any{{"world": "world", "loadedChunks": 957, "entities": 412, "tileEntities": 319, "players": 1,
				"entityTypes": map[string]int{"minecraft:zombie": 28}}},
			"jvm": map[string]any{
				"heap":    map[string]any{"usedBytes": 100, "committedBytes": 200, "maxBytes": 400},
				"nonHeap": map[string]any{"usedBytes": 10, "committedBytes": 20},
				"pools":   []map[string]any{{"name": "G1 Old Gen", "type": "heap", "usedBytes": 50, "committedBytes": 60, "maxBytes": 400}},
				"gc":      []map[string]any{{"name": "G1 Young Generation", "count": 19, "timeMillis": 647}},
				"threads": map[string]any{"live": 175, "daemon": 117, "peak": 183}, "classesLoaded": 56358,
			},
		}
	case path == "/worlds":
		body = []map[string]any{{"name": "world"}}
	case path == "/worlds/world":
		body = map[string]any{"name": "world", "environment": "normal", "time": map[string]any{"dayTime": 6000, "day": 718},
			"weather": map[string]any{"raining": true, "thundering": false}, "border": map[string]any{"size": 60000.0}}
	case path == "/players/online":
		body = map[string]any{"count": 1, "max": 20, "players": []map[string]any{{"uuid": "u-steve", "name": "Steve", "world": "world",
			"gameMode": "survival", "health": 18.5, "food": 20, "level": 7, "ping": 55, "idleSeconds": 3,
			"location": map[string]any{"world": "world", "x": 10.5, "y": 64, "z": -3}}}}
	case path == "/server/system":
		body = map[string]any{"cpus": 8, "loadAverage": []float64{1.5, 1, 0.5},
			"memory":    map[string]any{"totalBytes": 1000, "availableBytes": 400, "swapTotalBytes": nil},
			"disk":      map[string]any{"path": "/data", "totalBytes": 500, "usableBytes": 200},
			"container": map[string]any{"memoryLimitBytes": 900, "memoryUsedBytes": 300, "cpuLimit": 4.0},
			"openFiles": 120, "maxOpenFiles": 1024, "processResidentBytes": 700,
			"java": map[string]any{"name": "OpenJDK", "vendor": "Eclipse Adoptium", "version": "25.0.1"}}
	case path == "/server/logs":
		body = map[string]any{"lines": f.logs}
	case path == "/server/plugins":
		body = []map[string]any{{"name": "WorldAPI", "version": "3.2.0", "enabled": true}, {"name": "Broken", "version": "1.0", "enabled": false}}
	case path == "/server/bans":
		body = map[string]any{"players": []map[string]any{{"target": "Griefer"}}, "ips": []map[string]any{}}
	case path == "/server/ops":
		body = map[string]any{"players": []map[string]any{{"uuid": "u-steve"}}}
	case path == "/server/whitelist":
		body = map[string]any{"enabled": false, "players": []map[string]any{}}
	case path == "/worlds/world/entities/summary":
		body = map[string]any{"droppedItems": 12, "droppedItemCount": 300, "villagers": 40, "named": 2, "tamed": 5,
			"topChunks": []map[string]any{{"x": 3, "z": -7, "entities": 90, "tileEntities": 30, "hoppers": 12}}}
	case path == "/players":
		body = map[string]any{"items": []map[string]any{
			{"uuid": "u-steve", "name": "Steve", "firstPlayed": "2026-10-10T16:48:21Z", "lastSeen": "2026-10-19T15:00:00Z"},
			{"uuid": "u-alex", "name": "Alex", "firstPlayed": "2026-10-11T10:00:00Z", "lastSeen": "2026-10-18T10:00:00Z"},
		}, "nextCursor": nil}
	case path == "/leaderboards/custom:play_time":
		body = board(map[string]float64{"Steve": 72000, "Alex": 36000})
	case path == "/leaderboards/travel":
		body = board(map[string]float64{"Steve": 1.5})
	case strings.HasPrefix(path, "/leaderboards/"):
		body = board(map[string]float64{"Steve": 3, "Alex": 2})
	case path == "/extensions":
		body = []map[string]any{{"namespace": "quests", "topics": []map[string]any{
			{"topic": "ext.quests.quest.completed", "recorded": true}, {"topic": "ext.quests.quest.started", "recorded": false}}}}
	case path == "/events":
		body = f.eventPage(r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

func (f *fakeWorldAPI) eventPage(r *http.Request) map[string]any {
	q := r.URL.Query()
	items := []map[string]any{}
	if q.Get("order") == "newest" {
		if len(f.events) > 0 {
			items = append(items, f.events[len(f.events)-1])
		}
		return map[string]any{"items": items}
	}
	for _, e := range f.events {
		if q.Get("from") == "" || e["atMillis"].(int64) >= parseInt(q.Get("from")) {
			items = append(items, e)
		}
	}
	return map[string]any{"items": items, "nextCursor": nil}
}

func parseInt(s string) int64 {
	var n int64
	for _, c := range s {
		n = n*10 + int64(c-'0')
	}
	return n
}

func board(values map[string]float64) map[string]any {
	entries := []map[string]any{}
	for name, value := range values {
		entries = append(entries, map[string]any{"player": map[string]any{"uuid": "u-" + strings.ToLower(name), "name": name}, "value": value})
	}
	return map[string]any{"total": len(entries), "entries": entries}
}

func newFixture(t *testing.T, opts Options) (*fakeWorldAPI, *Target, *prometheus.CounterVec) {
	t.Helper()
	fake := &fakeWorldAPI{forbidden: map[string]bool{}, requests: map[string]int{}}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "errors_total"}, []string{"target", "endpoint"})
	client := worldapi.NewClient(server.URL+"/api/v1", "", "test", server.Client())
	return fake, NewTarget("test", client, opts, errorsTotal), errorsTotal
}

func scrapeText(t *testing.T, target *Target) string {
	t.Helper()
	registry := prometheus.NewRegistry()
	registry.MustRegister(target.Scrape(context.Background()))
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var out strings.Builder
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			labels := []string{}
			for _, pair := range metric.GetLabel() {
				labels = append(labels, pair.GetName()+"="+pair.GetValue())
			}
			value := metric.GetGauge().GetValue() + metric.GetCounter().GetValue()
			out.WriteString(family.GetName() + "{" + strings.Join(labels, ",") + "} " + jsonNumber(value) + "\n")
		}
	}
	return out.String()
}

// scrapeFilled scrapes until every statistic had its turn in the batched refresh.
func scrapeFilled(t *testing.T, target *Target) string {
	t.Helper()
	text := ""
	for range 8 {
		text = scrapeText(t, target)
	}
	return text
}

func jsonNumber(v float64) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func expectLines(t *testing.T, text string, lines ...string) {
	t.Helper()
	for _, line := range lines {
		if !strings.Contains(text, line+"\n") {
			t.Errorf("missing %q in\n%s", line, text)
		}
	}
}

func TestScrapeReadsTheServer(t *testing.T) {
	_, target, _ := newFixture(t, Options{Players: true, StatsInterval: time.Minute})
	text := scrapeFilled(t, target)
	expectLines(t, text,
		"minecraft_up{} 1",
		"minecraft_info{api_version=26.3.build.205-stable,brand=Paper,build=205,minecraft_version=26.3,worldapi_version=3.2.0} 1",
		"minecraft_players_online{} 1",
		"minecraft_datapacks_enabled{} 1",
		"minecraft_tps{window=1m} 19.5",
		"minecraft_tick_duration_seconds{quantile=0.95} 0.0125",
		"minecraft_tick_duration_seconds{quantile=1} 0.022",
		"minecraft_cpu_usage_ratio{scope=process} 0.5",
		"minecraft_jvm_memory_max_bytes{area=heap} 400",
		"minecraft_jvm_gc_seconds_total{gc=G1 Young Generation} 0.647",
		"minecraft_world_entities_by_type{type=minecraft:zombie,world=world} 28",
		"minecraft_world_days_total{world=world} 718",
		"minecraft_world_raining{world=world} 1",
		"minecraft_player_ping_seconds{player=Steve} 0.055",
		"minecraft_player_info{gamemode=survival,player=Steve,uuid=u-steve,world=world} 1",
		"minecraft_player_play_time_seconds_total{player=Alex} 1800",
		"minecraft_player_distance_meters_total{player=Steve} 1500",
		"minecraft_player_advancements{player=Steve} 3",
		"minecraft_players_known{} 2",
		"minecraft_player_last_seen_timestamp_seconds{player=Alex} 1792317600",
	)
	if strings.Contains(text, "minecraft_world_entities{world=world} 0") {
		t.Error("world entities read as zero")
	}
}

func TestWithoutPlayersStatisticsAddUp(t *testing.T) {
	_, target, _ := newFixture(t, Options{Players: false, StatsInterval: time.Minute})
	text := scrapeFilled(t, target)
	expectLines(t, text, "minecraft_player_play_time_seconds_total{} 5400", "minecraft_player_deaths_total{} 5", "minecraft_players_known{} 2")
	for _, leak := range []string{"player=", "minecraft_player_ping_seconds", "minecraft_player_first_join"} {
		if strings.Contains(text, leak) {
			t.Errorf("found %q with players off", leak)
		}
	}
}

func TestEventsCountFromTheExportersStart(t *testing.T) {
	fake, target, _ := newFixture(t, Options{Players: true, StatsInterval: time.Minute})
	fake.addEvent(1, "player.death", map[string]any{"cause": "minecraft:fall"})

	first := scrapeText(t, target)
	expectLines(t, first, "minecraft_events_total{type=player.death} 0", "minecraft_first_joins_total{} 0",
		"minecraft_events_total{type=ext.quests.quest.completed} 0", "minecraft_advancements_total{frame=goal} 0")
	if strings.Contains(first, "phase_two") {
		t.Error("seeded an event the plugin doesn't record")
	}

	fake.addEvent(2, "player.death", map[string]any{"cause": "minecraft:mob_attack", "killer": map[string]any{"type": "minecraft:zombie"}})
	fake.addEvent(3, "player.advancement", map[string]any{"frame": "challenge"})
	fake.addEvent(4, "player.join", map[string]any{"firstJoin": true})
	fake.addEvent(5, "ext.quests.quest.completed", map[string]any{})
	second := scrapeText(t, target)
	expectLines(t, second,
		"minecraft_events_total{type=player.death} 1",
		"minecraft_events_total{type=ext.quests.quest.completed} 1",
		"minecraft_deaths_total{cause=minecraft:mob_attack,killer=minecraft:zombie} 1",
		"minecraft_advancements_total{frame=challenge} 1",
		"minecraft_first_joins_total{} 1",
	)

	third := scrapeText(t, target)
	expectLines(t, third, "minecraft_events_total{type=player.death} 1")
}

func TestStatisticsStayCached(t *testing.T) {
	fake, target, _ := newFixture(t, Options{Players: true, StatsInterval: time.Hour})
	scrapeFilled(t, target)
	scrapeFilled(t, target)
	if n := fake.requests["/leaderboards/custom:deaths"]; n != 1 {
		t.Errorf("read a leaderboard %d times within the interval, want 1", n)
	}
}

func TestStatisticsRefreshInBatches(t *testing.T) {
	fake, target, _ := newFixture(t, Options{Players: true, StatsInterval: time.Hour})
	scrapeText(t, target)
	read := fake.requests["/players"]
	for path, n := range fake.requests {
		if strings.HasPrefix(path, "/leaderboards/") {
			read += n
		}
	}
	if read != slowBatch {
		t.Errorf("one scrape read %d statistics, want %d", read, slowBatch)
	}
}

func TestForbiddenEndpointsLeaveTheRest(t *testing.T) {
	fake, target, errorsTotal := newFixture(t, Options{Players: true, StatsInterval: time.Minute})
	fake.forbidden["/events"] = true
	fake.forbidden["/leaderboards"] = true
	text := scrapeText(t, target)
	expectLines(t, text, "minecraft_up{} 1", "minecraft_tps{window=1m} 19.5")
	if strings.Contains(text, "minecraft_events_total") || strings.Contains(text, "minecraft_player_deaths_total") {
		t.Error("exported metrics of forbidden endpoints")
	}
	if n := testutil.ToFloat64(errorsTotal.WithLabelValues("test", "events")); n != 0 {
		t.Errorf("a refusal counted as %v errors, want none", n)
	}
	expectLines(t, text, "minecraft_exporter_endpoint_allowed{endpoint=events} 0", "minecraft_exporter_endpoint_allowed{endpoint=server} 1")
	scrapeText(t, target)
	if n := fake.requests["/events"]; n != 1 {
		t.Errorf("asked a refused endpoint %d times within %v, want once", n, deniedFor)
	}
}

func TestDownServerReportsUpZero(t *testing.T) {
	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "errors_total"}, []string{"target", "endpoint"})
	client := worldapi.NewClient("http://127.0.0.1:1/api/v1", "", "test", &http.Client{Timeout: time.Second})
	target := NewTarget("down", client, Options{Players: true, StatsInterval: time.Minute}, errorsTotal)
	expectLines(t, scrapeText(t, target), "minecraft_up{} 0")
}

func TestTokenUnlocksPrivateData(t *testing.T) {
	fake, target, _ := newFixture(t, Options{Players: true, StatsInterval: time.Minute, Token: true})
	fake.addEvent(1, "player.join", map[string]any{})
	fake.addLog("2026-10-19T10:00:00Z", "info", "Minecraft", "old line")
	scrapeFilled(t, target)

	fake.addEvent(2, "player.chat", map[string]any{"message": "hi"})
	fake.addLog("2026-10-19T10:00:00Z", "warn", "Minecraft", "same millisecond, new line")
	fake.addLog("2026-10-19T10:00:01Z", "error", "Broken", "boom")
	fake.addLog("2026-10-19T10:00:01Z", "info", "Minecraft", "done")
	text := scrapeText(t, target)
	expectLines(t, text,
		"minecraft_host_load_average{window=5m} 1",
		"minecraft_host_memory_available_bytes{} 400",
		"minecraft_disk_free_bytes{} 200",
		"minecraft_container_cpu_limit{} 4",
		"minecraft_process_resident_memory_bytes{} 700",
		"minecraft_java_info{name=OpenJDK,vendor=Eclipse Adoptium,version=25.0.1} 1",
		"minecraft_plugin_enabled{plugin=Broken,version=1.0} 0",
		"minecraft_bans{kind=player} 1",
		"minecraft_operators{} 1",
		"minecraft_world_villagers{world=world} 40",
		"minecraft_world_dropped_item_count{world=world} 300",
		"minecraft_chunk_hoppers{chunk=3,-7,world=world} 12",
		"minecraft_player_position_blocks{axis=z,player=Steve,world=world} -3",
		"minecraft_player_chat_messages_total{player=Steve} 1",
		"minecraft_log_messages_total{level=warn} 1",
		"minecraft_log_messages_total{level=info} 1",
		"minecraft_log_problems_total{level=error,logger=Broken} 1",
	)
	if strings.Contains(text, "swap_total") {
		t.Error("exported a value the host doesn't report")
	}
	expectLines(t, scrapeText(t, target), "minecraft_log_messages_total{level=info} 1")
}

func TestWithoutTokenPrivateDataStaysAway(t *testing.T) {
	fake, target, errorsTotal := newFixture(t, Options{Players: true, StatsInterval: time.Minute})
	for _, path := range []string{"/server/system", "/server/logs", "/server/plugins", "/server/bans", "/worlds/world/entities"} {
		fake.forbidden[path] = true
	}
	text := scrapeFilled(t, target)
	for _, metric := range []string{"minecraft_host_", "minecraft_log_", "minecraft_plugin_", "minecraft_bans", "minecraft_world_villagers"} {
		if strings.Contains(text, metric) {
			t.Errorf("exported %s without its permission", metric)
		}
	}
	expectLines(t, text, "minecraft_exporter_endpoint_allowed{endpoint=server/system} 0")
	for _, endpoint := range []string{"server/system", "server/logs", "server/plugins"} {
		if n := testutil.ToFloat64(errorsTotal.WithLabelValues("test", endpoint)); n != 0 {
			t.Errorf("%s: refusals counted as %v errors", endpoint, n)
		}
	}
}
