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
	flag.StringVar(&configPath, "config", "local.yaml", "config file path")
	flag.Parse()
}

// App is main application struct
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
		//a.initCustomConfigs,
	}
	for _, f := range inits {
		if err := f(ctx); err != nil {
			return err
		}
	}

	return nil
}

// initConfig initializing config
func (a *App) initConfig(_ context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		slog.Error("failed to load config file")
		return err
	}
	return nil
}

// initServiceProvider initializing service provider
func (a *App) initServiceProvider(_ context.Context) error {
	a.serviceProvider = newServiceProvider()
	slog.Debug("Config vars",
		"loadBalancer", a.serviceProvider.LoadBalancerConfig(),
		"backendConfig", a.serviceProvider.BackendConfig(),
	)
	return nil
}

// initHTTPServer initializing http server and register handlers
func (a *App) initHTTPServer(ctx context.Context) error {

	// Register endpoints
	a.serviceProvider.LoadBalancerImpl(ctx).Check(ctx)
	a.serviceProvider.LoadBalancerImpl(ctx).GetClients(ctx)
	a.serviceProvider.LoadBalancerImpl(ctx).Add(ctx)
	a.serviceProvider.LoadBalancerImpl(ctx).Get(ctx)
	a.serviceProvider.LoadBalancerImpl(ctx).Update(ctx)
	a.serviceProvider.LoadBalancerImpl(ctx).Delete(ctx)

	err := a.serviceProvider.RateLimiter(ctx).CheckAll(ctx)
	if err != nil {
		slog.Error("failed to check all rate limiter clients",
			"error", err.Error(),
		)
	}

	a.httpServer = &http.Server{
		Addr:    a.serviceProvider.LoadBalancerConfig().Address(),
		Handler: a.serviceProvider.LoadBalancerImpl(ctx),
	}
	return nil
}

// initHealthChecker initializing health checker service
func (a *App) initHealthChecker(ctx context.Context) error {
	a.healthChecker = healthChecker.NewHealthChecker(a.serviceProvider.Backends(ctx), a.serviceProvider.HealthCheckerConfig().Interval())
	return nil
}

// runHTTPServer start http server
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
