package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"realtime-dashboard/internal/aggregator"
	"realtime-dashboard/internal/collector"
	"realtime-dashboard/internal/hub"
	"realtime-dashboard/internal/metrics"
	"realtime-dashboard/internal/server"
	"sync"
	"syscall"
	"time"
)

const (
	eventBuffer   = 256
	headerTimeout = time.Second * 5
	serverTimeout = time.Second * 5
)

type config struct {
	port            string
	tick            time.Duration
	period          time.Duration
	runtimeInterval time.Duration
	webDir          string
}

func parseFlags() config {
	var port, dir string
	var tick, period, interval time.Duration

	flag.StringVar(&port, "port", ":9090", "where to listen as example :8080")
	flag.DurationVar(&tick, "tick", time.Second, "rate of snapshots being sent to server")
	flag.DurationVar(&period, "window", time.Second*60, "how far back the stats look")
	flag.DurationVar(&interval, "runtime", time.Second, "how often internal usage is calculated (heap and goroutine measure)")
	flag.StringVar(&dir, "web", "web", "local folder for dashboard page default is web")
	flag.Parse()

	return config{
		port:            port,
		tick:            tick,
		period:          period,
		runtimeInterval: interval,
		webDir:          dir,
	}
}

func run(ctx context.Context, cfg config) error {
	events := make(chan metrics.Event, eventBuffer)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	h := hub.NewHub()
	agg := aggregator.NewAggregator(events, cfg.tick, cfg.period, h.Broadcast)
	collectorSource := collector.NewRuntimeSource(cfg.runtimeInterval)
	wikimedia := collector.NewWikimediaSource()
	var waitGroup sync.WaitGroup

	waitGroup.Go(func() {
		h.Run(ctx)
	})

	waitGroup.Go(func() {
		agg.Run(ctx)
	})

	waitGroup.Go(func() {
		err := collectorSource.Run(ctx, events)
		if err != nil {
			log.Printf("collector source stopped: %v", err)
		}
	})

	waitGroup.Go(func() {
		err := wikimedia.Run(ctx, events)
		if err != nil {
			log.Printf("wikimedia source stopped: %v", err)
		}
	})
	srv := server.NewServer(h, events, agg, cfg.period, cfg.webDir)
	httpServer := &http.Server{
		Handler:           srv.Routes(),
		Addr:              cfg.port,
		ReadHeaderTimeout: headerTimeout,
	}
	errChannel := make(chan error, 1)
	log.Printf("listening on http://localhost%s", cfg.port)
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errChannel <- err
			}
		}
	}()

	var serverErr error
	select {
	case <-ctx.Done():
	case serverErr = <-errChannel:
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), serverTimeout)
	defer shutdownCancel()
	err := httpServer.Shutdown(shutdownCtx)
	if err != nil {
		log.Printf("server shutdown error occurred %v", err)
	}
	cancel()
	waitGroup.Wait()

	return serverErr
}

func main() {
	cfg := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx, cfg)
	if err != nil {
		log.Fatalf("could not run: %v", err)
	}
}
