package loadBalancer

import (
	"context"
	"encoding/json"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"io"
	"log/slog"
	"net/http"
)

func (i *Implementation) Add(ctx context.Context) {
	i.mux.HandleFunc("/clients/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
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
}
