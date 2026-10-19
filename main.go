// minecraft-exporter serves Prometheus metrics of Minecraft servers that run WorldAPI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mairwunnx/minecraft-exporter/internal/collector"
	"github.com/mairwunnx/minecraft-exporter/internal/config"
	"github.com/mairwunnx/minecraft-exporter/internal/worldapi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// version is stamped by the release build.
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		worldapiURL   = flag.String("worldapi.url", env("WORLDAPI_URL", ""), "WorldAPI to read at /metrics, like http://localhost:8110/api/v1 (WORLDAPI_URL)")
		token         = flag.String("worldapi.token", env("WORLDAPI_TOKEN", ""), "WorldAPI token; anonymous reads without one (WORLDAPI_TOKEN)")
		tokenFile     = flag.String("worldapi.token-file", env("WORLDAPI_TOKEN_FILE", ""), "File that holds the token (WORLDAPI_TOKEN_FILE)")
		targetsFile   = flag.String("targets", env("EXPORTER_TARGETS", ""), "YAML file of servers for /probe?target=<name> (EXPORTER_TARGETS)")
		listen        = flag.String("listen", env("EXPORTER_LISTEN", ":10059"), "Address to serve metrics on (EXPORTER_LISTEN)")
		players       = flag.Bool("players", envBool("EXPORTER_PLAYERS", true), "Label per-player metrics with the player; false adds them up (EXPORTER_PLAYERS)")
		statsInterval = flag.String("stats-interval", env("EXPORTER_STATS_INTERVAL", "60s"), "How long player statistics stay cached (EXPORTER_STATS_INTERVAL)")
		timeout       = flag.String("timeout", env("EXPORTER_TIMEOUT", "10s"), "Longest a scrape may take, unless Prometheus asks for less (EXPORTER_TIMEOUT)")
		showVersion   = flag.Bool("version", false, "Print the version and exit")
	)
	flag.Parse()
	if *showVersion {
		fmt.Println("minecraft-exporter", version)
		return nil
	}

	interval, err := config.Duration(*statsInterval)
	if err != nil {
		return fmt.Errorf("stats interval: %w", err)
	}
	scrapeTimeout, err := config.Duration(*timeout)
	if err != nil {
		return fmt.Errorf("timeout: %w", err)
	}

	errorsTotal := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "minecraft_exporter_request_errors_total",
		Help: "WorldAPI requests that failed, by target and endpoint; a missing permission counts too",
	}, []string{"target", "endpoint"})
	own := prometheus.NewRegistry()
	own.MustRegister(
		errorsTotal,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "minecraft_exporter_build_info",
			Help:        "The exporter's version",
			ConstLabels: prometheus.Labels{"version": version, "goversion": runtime.Version()},
		}, func() float64 { return 1 }),
	)

	httpClient := &http.Client{Timeout: scrapeTimeout}
	userAgent := "minecraft-exporter/" + version
	newTarget := func(name, rawURL, token, tokenFile string, perPlayer bool) (*collector.Target, error) {
		if err := config.CheckURL(rawURL); err != nil {
			return nil, err
		}
		secret, err := config.ReadToken(token, tokenFile)
		if err != nil {
			return nil, fmt.Errorf("token file: %w", err)
		}
		client := worldapi.NewClient(rawURL, secret, userAgent, httpClient)
		return collector.NewTarget(name, client, collector.Options{Players: perPlayer, StatsInterval: interval}, errorsTotal), nil
	}

	var primary *collector.Target
	if *worldapiURL != "" {
		if primary, err = newTarget("default", *worldapiURL, *token, *tokenFile, *players); err != nil {
			return fmt.Errorf("WORLDAPI_URL: %w", err)
		}
	}
	probes := map[string]*collector.Target{}
	if *targetsFile != "" {
		file, err := config.LoadFile(*targetsFile)
		if err != nil {
			return err
		}
		for name, t := range file.Targets {
			perPlayer := *players
			if t.Players != nil {
				perPlayer = *t.Players
			}
			if probes[name], err = newTarget(name, t.URL, t.Token, t.TokenFile, perPlayer); err != nil {
				return fmt.Errorf("target %q: %w", name, err)
			}
		}
	}
	if primary == nil && len(probes) == 0 {
		return errors.New("nothing to read: set WORLDAPI_URL, or EXPORTER_TARGETS for /probe")
	}

	serve := func(w http.ResponseWriter, r *http.Request, target *collector.Target, gatherers ...prometheus.Gatherer) {
		ctx, cancel := context.WithTimeout(r.Context(), scrapeBudget(r, scrapeTimeout))
		defer cancel()
		registry := prometheus.NewRegistry()
		registry.MustRegister(target.Scrape(ctx))
		promhttp.HandlerFor(append(prometheus.Gatherers{registry}, gatherers...), promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		if primary == nil {
			promhttp.HandlerFor(own, promhttp.HandlerOpts{}).ServeHTTP(w, r)
			return
		}
		serve(w, r, primary, own)
	})
	mux.HandleFunc("GET /probe", func(w http.ResponseWriter, r *http.Request) {
		target, ok := probes[r.URL.Query().Get("target")]
		if !ok {
			http.Error(w, "unknown target: name one from the targets file", http.StatusNotFound)
			return
		}
		serve(w, r, target)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html><title>minecraft-exporter</title><h1>minecraft-exporter %s</h1>
<p><a href="metrics">/metrics</a> · <a href="https://github.com/MairwunNx/minecraft-exporter">GitHub</a></p>`, version)
	})

	server := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()

	slog.Info("minecraft-exporter "+version+" listening", "address", *listen, "worldapi", *worldapiURL, "targets", len(probes), "players", *players)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// scrapeBudget leaves half a second of Prometheus' own timeout for sending the answer.
func scrapeBudget(r *http.Request, limit time.Duration) time.Duration {
	if header := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); header != "" {
		if seconds, err := strconv.ParseFloat(header, 64); err == nil && seconds > 1 {
			return min(limit, time.Duration((seconds-0.5)*float64(time.Second)))
		}
	}
	return limit
}

func env(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	slog.Warn("not a boolean, using the default", "variable", name, "value", value, "default", fallback)
	return fallback
}
