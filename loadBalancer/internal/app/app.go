package app

import (
	"context"
	"errors"
	"flag"
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service/healthChecker"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", "local.yaml", "config file path")
}

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

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initServiceProvider,
		a.initHTTPServer,
		a.initHealthChecker,
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
	return err
}

func (a *App) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	slog.Debug("Config vars",
		"loadBalancer", a.serviceProvider.LoadBalancerConfig(),
		"backendConfig", a.serviceProvider.BackendConfig(),
	)
	return nil
}

func (a *App) initHTTPServer(_ context.Context) error {
	const numWorkers = 5
	type Job struct {
		w    http.ResponseWriter
		r    *http.Request
		done chan struct{}
	}

	mux := http.NewServeMux()
	jobs := make(chan Job, 100)

	for i := 0; i < numWorkers; i++ {
		go func(id int, jobs <-chan Job) {
			for job := range jobs {
				slog.Info("Worker",
					"id", id,
					"url", job.r.RemoteAddr,
				)
				a.serviceProvider.Proxy().ServeHTTP(job.w, job.r)
				job.done <- struct{}{}
			}
		}(i, jobs)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jobDone := make(chan struct{})
		select {
		case jobs <- Job{w, r, jobDone}:
			<-jobDone
		default:
			http.Error(w, "Servers unavailable", http.StatusServiceUnavailable)
		}
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
