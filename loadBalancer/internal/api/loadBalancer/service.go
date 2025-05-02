package loadBalancer

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"net/http"
)

type Implementation struct {
	mux   *http.ServeMux
	proxy http.Handler

	rateLimiter service.RateLimiter
	workerPool  service.WorkerPool
}

func NewImplementation(proxy http.Handler, rateLimiter service.RateLimiter, workerPool service.WorkerPool) *Implementation {
	return &Implementation{
		proxy:       proxy,
		mux:         http.NewServeMux(),
		rateLimiter: rateLimiter,
		workerPool:  workerPool,
	}
}

func (i *Implementation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.mux.ServeHTTP(w, r)
}
