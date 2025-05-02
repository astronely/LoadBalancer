package healthChecker

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"github.com/astronely/loadBalancer/loadBalancer/pkg/closer"
	"log/slog"
	"net/http"
	"time"
)

// HealthChecker struct of service
type HealthChecker struct {
	backends []service.Backend
	interval time.Duration
	client   *http.Client
}

func NewHealthChecker(backends []service.Backend, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		backends: backends,
		interval: interval,
		client:   &http.Client{Timeout: 2 * time.Second},
	}
}

// Run Health Checker - checking if backend available, if not - set alive to false, if yes - to true
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
					b.GetURL().String(), "is Alive")
			} else {
				slog.Info("HealthChecker", b.GetURL().String(), "is Dead")
			}
		}
	}
}

// check if backend available
func (h *HealthChecker) check(b service.Backend) bool {
	healthUrl := b.GetURL().String() + "/health"
	res, err := h.client.Get(healthUrl)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}
