// Package collector turns one WorldAPI's answers into Prometheus metrics, fresh on every scrape.
package collector

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
	"github.com/prometheus/client_golang/prometheus"
)

// Options say what a target exports.
type Options struct {
	// Players adds a player label to per-player metrics; without it they add up every player.
	Players bool
	// StatsInterval is how long statistics, which move slowly and cost a request each, stay cached.
	StatsInterval time.Duration
}

// Target is one Minecraft server behind WorldAPI. It keeps what must survive between scrapes:
// the event counters and the cached statistics.
type Target struct {
	Name   string
	client *worldapi.Client
	opts   Options
	errors *prometheus.CounterVec

	statsMu sync.Mutex
	stats   statsCache

	eventsMu sync.Mutex
	events   eventCounts
}

func NewTarget(name string, client *worldapi.Client, opts Options, errors *prometheus.CounterVec) *Target {
	return &Target{Name: name, client: client, opts: opts, errors: errors, events: newEventCounts()}
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
		serverErr   error
	)
	wg.Go(func() { server, serverErr = t.client.GetServer(ctx); t.failed("server", serverErr) })
	wg.Go(func() {
		var err error
		performance, err = t.client.GetPerformance(ctx)
		t.failed("server/performance", err)
	})
	wg.Go(func() {
		var err error
		worlds, err = t.client.GetWorlds(ctx)
		t.failed("worlds", err)
	})
	wg.Go(func() {
		var err error
		online, err = t.client.GetOnlinePlayers(ctx)
		t.failed("players/online", err)
	})
	wg.Go(func() { t.refreshStats(ctx) })
	wg.Go(func() { t.pollEvents(ctx) })
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
	t.collectStats(out)
	t.collectEvents(out)
	out.gauge("minecraft_exporter_scrape_duration_seconds", "How long reading WorldAPI took", time.Since(start).Seconds())
}

// failed counts a failed request. A missing permission counts too: the metrics it would give stay away.
func (t *Target) failed(endpoint string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	t.errors.WithLabelValues(t.Name, endpoint).Inc()
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
