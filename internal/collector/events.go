package collector

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
)

// Event types every server records, and advancement frames. Their counters start at zero, so the first
// event shows as an increase; other plugins' recorded events join them at the first poll.
var (
	seededTypes  = []string{"player.join", "player.leave", "player.kick", "player.death", "player.advancement", "player.chat"}
	seededFrames = []string{"task", "goal", "challenge"}
)

const eventPage = 200

// eventCounts are counters fed by WorldAPI's event history. They count from the exporter's start:
// the history before it is left alone, like any counter that starts with its process.
type eventCounts struct {
	started bool
	lastID  int64
	lastAt  int64

	byType       map[string]float64
	deaths       map[[2]string]float64 // cause, killer
	advancements map[string]float64    // frame
	firstJoins   float64
}

func newEventCounts() eventCounts {
	counts := eventCounts{byType: map[string]float64{}, deaths: map[[2]string]float64{}, advancements: map[string]float64{}}
	for _, kind := range seededTypes {
		counts.byType[kind] = 0
	}
	for _, frame := range seededFrames {
		counts.advancements[frame] = 0
	}
	return counts
}

// pollEvents reads what was recorded since the last poll. The first poll only marks where to start.
func (t *Target) pollEvents(ctx context.Context) {
	t.eventsMu.Lock()
	defer t.eventsMu.Unlock()
	e := &t.events

	if !e.started {
		page, err := t.client.GetEvents(ctx, url.Values{"limit": {"1"}, "order": {"newest"}})
		t.failed("events", err)
		if err != nil {
			return
		}
		if len(page.Items) > 0 {
			e.lastID, e.lastAt = page.Items[0].ID, page.Items[0].AtMillis
		}
		e.started = true
		extensions, err := t.client.GetExtensions(ctx)
		t.failed("extensions", err)
		for _, extension := range extensions {
			for _, topic := range extension.Topics {
				if _, seen := e.byType[topic.Topic]; topic.Recorded && !seen {
					e.byType[topic.Topic] = 0
				}
			}
		}
		return
	}

	query := url.Values{"limit": {strconv.Itoa(eventPage)}, "order": {"oldest"}, "from": {strconv.FormatInt(e.lastAt, 10)}}
	for range 50 {
		page, err := t.client.GetEvents(ctx, query)
		t.failed("events", err)
		if err != nil {
			return
		}
		for _, event := range page.Items {
			if event.ID <= e.lastID {
				continue
			}
			e.count(event)
			e.lastID = event.ID
			e.lastAt = max(e.lastAt, event.AtMillis)
		}
		if len(page.Items) < eventPage || page.NextCursor == nil {
			return
		}
		query.Set("cursor", *page.NextCursor)
	}
}

func (e *eventCounts) count(event worldapi.Event) {
	e.byType[event.Type]++
	switch event.Type {
	case "player.death":
		var data struct {
			Cause  string `json:"cause"`
			Killer *struct {
				Type string `json:"type"`
			} `json:"killer"`
		}
		_ = json.Unmarshal(event.Data, &data)
		killer := ""
		if data.Killer != nil {
			killer = data.Killer.Type
		}
		e.deaths[[2]string{data.Cause, killer}]++
	case "player.advancement":
		var data struct {
			Frame string `json:"frame"`
		}
		_ = json.Unmarshal(event.Data, &data)
		e.advancements[data.Frame]++
	case "player.join":
		var data struct {
			FirstJoin bool `json:"firstJoin"`
		}
		_ = json.Unmarshal(event.Data, &data)
		if data.FirstJoin {
			e.firstJoins++
		}
	}
}

func (t *Target) collectEvents(out *emitter) {
	t.eventsMu.Lock()
	defer t.eventsMu.Unlock()
	e := &t.events
	if !e.started {
		return
	}
	for kind, n := range e.byType {
		out.counter("minecraft_events_total", "Events WorldAPI recorded since the exporter started, by type, other plugins' included", n, "type", kind)
	}
	for key, n := range e.deaths {
		out.counter("minecraft_deaths_total", "Player deaths since the exporter started, by cause and killer", n, "cause", key[0], "killer", key[1])
	}
	for frame, n := range e.advancements {
		out.counter("minecraft_advancements_total", "Advancements players made since the exporter started, by frame: task, goal, or challenge", n, "frame", frame)
	}
	out.counter("minecraft_first_joins_total", "Players who joined for the first time since the exporter started", e.firstJoins)
}
