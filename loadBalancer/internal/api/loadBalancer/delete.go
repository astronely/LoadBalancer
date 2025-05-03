package loadBalancer

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// Delete handler deletes user rate limits from Redis
func (i *Implementation) Delete(ctx context.Context) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := strings.Split(r.URL.Path, "/")
		if len(urlPath) < 3 || urlPath[2] == "" {
			http.Error(w, "Id not found", http.StatusNotFound)
			return
		}

		err := i.rateLimiter.Delete(ctx, urlPath[2])
		if err != nil {
			if err.Error() == "redis: nil" {
				http.Error(w, "Client not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Rate limiter error", http.StatusServiceUnavailable)
			return
		}
	})

	wrappedHandler := i.rateLimiter.Middleware(handler, func(r *http.Request) string {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		return host
	})
	i.mux.Handle("DELETE /clients/", wrappedHandler)
}
