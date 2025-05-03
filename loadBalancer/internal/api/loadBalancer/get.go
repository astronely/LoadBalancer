package loadBalancer

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

// Get handler returns user rate limits from Redis
func (i *Implementation) Get(ctx context.Context) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedUrlPath := strings.Split(r.URL.Path, "/")
		if len(parsedUrlPath) < 3 || parsedUrlPath[1] == "" {
			http.Error(w, "Id not found", http.StatusNotFound)
			return
		}

		res, err := i.rateLimiter.Get(ctx, parsedUrlPath[2])
		if err != nil {
			if err.Error() == "redis: nil" {
				http.Error(w, "Client not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Rate limiter error", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	wrappedHandler := i.rateLimiter.Middleware(handler, func(r *http.Request) string {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		return host
	})
	i.mux.Handle("GET /clients/", wrappedHandler)
}
