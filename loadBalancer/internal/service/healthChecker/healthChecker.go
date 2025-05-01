package healthChecker

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/backend"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"log/slog"
	"net/http"
	"time"
)

type HealthChecker struct {
	backends []*backend.Backend
	interval time.Duration
	client   *http.Client
}

func NewHealthChecker(backends []*backend.Backend, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		backends: backends,
		interval: interval,
		client:   &http.Client{Timeout: 2 * time.Second}, // TODO: To config
	}
}

func (h *HealthChecker) Run() {
	ticker := time.NewTicker(h.interval)
	closer.Add(func() error {
		slog.Info("health check server shutdown down gracefully")
		ticker.Stop()
		return nil
	})

	for range ticker.C {
		for _, b := range h.backends {
			alive := h.check(b)
			b.SetAlive(alive)
			if alive {
				slog.Info("HealthChecker",
					b.URL.String(), "is Alive")
			} else {
				slog.Info("HealthChecker", b.URL.String(), "is Dead")
			}
		}
	}
}

func (h *HealthChecker) check(b *backend.Backend) bool {
	healthUrl := b.URL.String() + "/health"
	res, err := h.client.Get(healthUrl)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}
