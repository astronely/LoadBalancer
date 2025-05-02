package loadBalancer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func (i *Implementation) Get(ctx context.Context) {
	i.mux.HandleFunc("/clients/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var info map[string]interface{}
		if err = json.Unmarshal(body, &info); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		val, ok := info["id"]
		if !ok {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		}

		res, err := i.rateLimiter.Get(ctx, val.(string))
		if err != nil {
			http.Error(w, "Rate limiter error", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})
}
