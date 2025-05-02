package loadBalancer

import (
	"context"
	"encoding/json"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/rateLimiter"
	"io"
	"net/http"
)

// Update handler updates user rate limits in Redis
func (i *Implementation) Update(ctx context.Context) {
	i.mux.HandleFunc("/clients/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
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
			return
		}

		convertedInfo := &rateLimiter.Info{
			ID:         info.ID,
			Capacity:   info.Capacity,
			RefillRate: info.RefillRate,
			Interval:   info.Interval,
		}

		err = i.rateLimiter.Update(ctx, convertedInfo)
		if err != nil {
			http.Error(w, "Failed to add rate limiter", http.StatusInternalServerError)
			return
		}
	})
}
