package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/workerPool"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/healthChecker"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "local.yaml", "config file path")
	flag.Parse()
}

// App - main app
type App struct {
	serviceProvider *serviceProvider
	httpServer      *http.Server
	healthChecker   *healthChecker.HealthChecker
}

func NewApp(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Run servers
func (a *App) Run(ctx context.Context) error {
	defer func() {
		slog.Info("shutting down the server...")
		closer.CloseAll()
		closer.Wait()
	}()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		err := a.runHTTPServer(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start http server: %v", err)
		}
	}()

	go func() {
		err := a.runHealthChecker(ctx)
		if err != nil {
			log.Fatalf("failed to start health checker: %v", err)
		}
	}()

	<-ctx.Done()
	return nil
}

// initDeps initializing dependencies
func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		a.initHTTPServer,
		a.initHealthChecker,
		a.initCustomConfigs,
	}
	for _, f := range inits {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) initConfig(_ context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config file")
		return err
	}
	return nil
}

func (a *App) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	slog.Debug("Config vars",
		"loadBalancer", a.serviceProvider.LoadBalancerConfig(),
		"backendConfig", a.serviceProvider.BackendConfig(),
	)
	return nil
}

func (a *App) initHTTPServer(ctx context.Context) error {
	mux := http.NewServeMux()
	wp := a.serviceProvider.WorkerPool()
	jobs := wp.Start()

	// Handler using WorkPool pattern
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*1)

		jobDone := make(chan struct{})
		select {
		case jobs <- workerPool.Job{W: w, R: r, Handler: a.serviceProvider.Proxy(), Done: jobDone}:
			<-jobDone
			cancel()
		case <-ctxWithTimeout.Done():
			http.Error(w, "Timeout, servers unavailable", http.StatusServiceUnavailable)
			cancel()
		}
	})

	// Handler return rate info for all clients
	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clients := a.serviceProvider.RateLimiter().Clients()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clients)
	})

	a.httpServer = &http.Server{
		Addr:    a.serviceProvider.LoadBalancerConfig().Address(),
		Handler: mux,
	}
	return nil
}

func (a *App) initHealthChecker(_ context.Context) error {
	a.healthChecker = healthChecker.NewHealthChecker(a.serviceProvider.Backends(), a.serviceProvider.HealthCheckerConfig().Interval())
	return nil
}

// initCustomConfigs initializing special config for VIP clients
func (a *App) initCustomConfigs(_ context.Context) error {
	a.serviceProvider.RateLimiter().SetClientConfig("127.0.0.1", a.serviceProvider.RateLimiterVipConfig())
	return nil
}

func (a *App) runHTTPServer(ctx context.Context) error {
	slog.Info("starting http server",
		"address", a.serviceProvider.LoadBalancerConfig().Address(),
	)

	closer.Add(func() error {
		slog.Info("HTTP server shutdown down gracefully")
		if err := a.httpServer.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	})

	err := a.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start http server",
			"error", err,
		)
		return err
	}
	return nil
}

func (a *App) runHealthChecker(_ context.Context) error {
	slog.Info("starting health checker")
	a.healthChecker.Run()
	return nil
}
