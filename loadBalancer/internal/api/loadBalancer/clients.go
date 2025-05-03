package loadBalancer

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
)

// GetClients handler return rate info for all clients
func (i *Implementation) GetClients(ctx context.Context) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clients := i.rateLimiter.Clients()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clients)
	})

	wrappedHandler := i.rateLimiter.Middleware(handler, func(r *http.Request) string {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		return host
	})
	i.mux.Handle("GET /clients", wrappedHandler)
}
