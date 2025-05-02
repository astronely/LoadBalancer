package loadBalancer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Delete handler deletes user rate limits from Redis
func (i *Implementation) Delete(ctx context.Context) {
	i.mux.HandleFunc("/clients/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
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

		err = i.rateLimiter.Delete(ctx, val.(string))
		if err != nil {
			http.Error(w, "Rate limiter error", http.StatusServiceUnavailable)
			return
		}
	})
}
