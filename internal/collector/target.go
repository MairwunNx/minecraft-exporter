// Package collector turns one WorldAPI's answers into Prometheus metrics, fresh on every scrape.
package collector

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
	"github.com/prometheus/client_golang/prometheus"
)

// Options say what a target exports.
type Options struct {
	// Players adds a player label to per-player metrics; without it they add up every player.
	Players bool
	// StatsInterval is how long slow data, like statistics, stays cached; each piece costs a request.
	StatsInterval time.Duration
	// Token says the exporter reads with a token, whose rate limit allows bigger batches.
	Token bool
}

// deniedFor is how long the exporter leaves alone an endpoint its token may not read.
const deniedFor = 10 * time.Minute

// Target is one Minecraft server behind WorldAPI. It keeps what must survive between scrapes:
// the event and log counters, the cached slow data, and which endpoints the token may read.
type Target struct {
	Name   string
	client *worldapi.Client
	opts   Options
	errors *prometheus.CounterVec

	slowMu sync.Mutex
	slow   slowCache

	eventsMu sync.Mutex
	events   eventCounts

	logsMu sync.Mutex
	logs   logCounts

	accessMu   sync.Mutex
	access     map[string]time.Time // endpoint → denied until; zero when allowed
	worlds     []string             // the worlds the last scrape saw, for per-world slow data
	namespaces []string             // plugins sharing data through WorldAPI
}

func NewTarget(name string, client *worldapi.Client, opts Options, errors *prometheus.CounterVec) *Target {
	return &Target{Name: name, client: client, opts: opts, errors: errors, events: newEventCounts(),
		logs: newLogCounts(), access: map[string]time.Time{}}
}

// Scrape returns a collector that reads the server once, within ctx.
func (t *Target) Scrape(ctx context.Context) prometheus.Collector {
	return &scrape{target: t, ctx: ctx}
}

type scrape struct {
	target *Target
	ctx    context.Context
}

// Describe sends nothing: the metrics depend on the server, so the registry checks them as they come.
func (s *scrape) Describe(chan<- *prometheus.Desc) {}

func (s *scrape) Collect(ch chan<- prometheus.Metric) {
	t, ctx, start := s.target, s.ctx, time.Now()
	out := &emitter{ch: ch}

	var (
		wg          sync.WaitGroup
		server      *worldapi.Server
		performance *worldapi.Performance
		worlds      []worldapi.World
		online      *worldapi.OnlinePlayers
		system      *worldapi.System
		serverErr   error
	)
	wg.Go(func() { server, serverErr = t.client.GetServer(ctx); t.done("server", serverErr) })
	wg.Go(func() {
		var err error
		performance, err = t.client.GetPerformance(ctx)
		t.done("server/performance", err)
	})
	wg.Go(func() {
		var err error
		if worlds, err = t.client.GetWorlds(ctx); t.done("worlds", err) {
			t.accessMu.Lock()
			t.worlds = t.worlds[:0]
			for _, w := range worlds {
				t.worlds = append(t.worlds, w.Name)
			}
			t.accessMu.Unlock()
		}
	})
	wg.Go(func() {
		var err error
		online, err = t.client.GetOnlinePlayers(ctx)
		t.done("players/online", err)
	})
	if t.allowed("server/system") {
		wg.Go(func() {
			var err error
			system, err = t.client.GetSystem(ctx)
			t.done("server/system", err)
		})
	}
	wg.Go(func() { t.refreshSlow(ctx) })
	wg.Go(func() { t.pollEvents(ctx) })
	wg.Go(func() { t.pollLogs(ctx) })
	wg.Wait()

	up := 0.0
	if serverErr == nil {
		up = 1
		collectServer(out, server)
	}
	out.gauge("minecraft_up", "1 when WorldAPI answered the last scrape, 0 when it didn't", up)
	if performance != nil {
		collectPerformance(out, performance)
	}
	if worlds != nil {
		collectWorlds(out, worlds)
	}
	if online != nil {
		collectOnline(out, online, t.opts.Players)
	}
	if system != nil {
		collectSystem(out, system)
	}
	t.collectStats(out)
	t.collectPrivate(out)
	t.collectEvents(out)
	t.collectLogs(out)
	t.collectAccess(out)
	out.gauge("minecraft_exporter_scrape_duration_seconds", "How long reading WorldAPI took", time.Since(start).Seconds())
}

// allowed says whether to call an endpoint: false for a while after WorldAPI refused the token.
func (t *Target) allowed(endpoint string) bool {
	t.accessMu.Lock()
	defer t.accessMu.Unlock()
	return time.Now().After(t.access[endpoint])
}

// done records how a request went and says whether it succeeded. A refusal (401, 403) marks the
// endpoint as closed to this token for a while; anything else that fails counts as an error.
func (t *Target) done(endpoint string, err error) bool {
	t.accessMu.Lock()
	defer t.accessMu.Unlock()
	var refused *worldapi.Error
	switch {
	case err == nil:
		t.access[endpoint] = time.Time{}
		return true
	case errors.As(err, &refused) && (refused.Status == http.StatusUnauthorized || refused.Status == http.StatusForbidden):
		t.access[endpoint] = time.Now().Add(deniedFor)
	case errors.Is(err, context.Canceled):
	default:
		t.errors.WithLabelValues(t.Name, endpoint).Inc()
	}
	return false
}

func (t *Target) collectAccess(out *emitter) {
	t.accessMu.Lock()
	defer t.accessMu.Unlock()
	for endpoint, until := range t.access {
		out.gauge("minecraft_exporter_endpoint_allowed",
			"1 when WorldAPI lets the exporter read an endpoint, 0 when the token lacks its permission", boolValue(until.IsZero()),
			"endpoint", endpoint)
	}
}

// emitter builds constant metrics, so every scrape reports the server as it is now.
type emitter struct {
	ch chan<- prometheus.Metric
}

func (e *emitter) emit(kind prometheus.ValueType, name, help string, value float64, labels ...string) {
	names, values := make([]string, 0, len(labels)/2), make([]string, 0, len(labels)/2)
	for i := 0; i+1 < len(labels); i += 2 {
		names, values = append(names, labels[i]), append(values, labels[i+1])
	}
	e.ch <- prometheus.MustNewConstMetric(prometheus.NewDesc(name, help, names, nil), kind, value, values...)
}

// gauge sends a gauge; labels go as name, value pairs.
func (e *emitter) gauge(name, help string, value float64, labels ...string) {
	e.emit(prometheus.GaugeValue, name, help, value, labels...)
}

// counter sends a counter; labels go as name, value pairs.
func (e *emitter) counter(name, help string, value float64, labels ...string) {
	e.emit(prometheus.CounterValue, name, help, value, labels...)
}
