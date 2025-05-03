package loadBalancer

import (
	"context"
	"encoding/json"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"io"
	"log/slog"
	"net"
	"net/http"
)

// Add handler add user rate limits
func (i *Implementation) Add(ctx context.Context) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var info rateLimiter.InfoFromBody
		if err = json.Unmarshal(body, &info); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			slog.Error("Invalid JSON payload",
				"json", string(body),
				"error", err.Error(),
			)
			return
		}

		convertedInfo := &rateLimiter.Info{
			ID:         info.ID,
			Capacity:   info.Capacity,
			RefillRate: info.RefillRate,
			Interval:   info.Interval,
		}

		err = i.rateLimiter.Add(ctx, convertedInfo)
		if err != nil {
			http.Error(w, "Failed to add rate limiter", http.StatusInternalServerError)
			return
		}
	})

	wrappedHandler := i.rateLimiter.Middleware(handler, func(r *http.Request) string {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		return host
	})

	i.mux.Handle("POST /clients", wrappedHandler)
}
