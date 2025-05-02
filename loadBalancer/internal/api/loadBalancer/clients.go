package loadBalancer

import (
	"context"
	"encoding/json"
	"net/http"
)

// GetClients handler return rate info for all clients
func (i *Implementation) GetClients(ctx context.Context) {
	i.mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clients := i.rateLimiter.Clients()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(clients)
	})
}
