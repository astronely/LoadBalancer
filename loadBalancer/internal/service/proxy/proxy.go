package proxy

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

// Proxy handler
type Proxy struct {
	loadBalancer service.LoadBalancer
}

func NewProxy(loadBalancer service.LoadBalancer) http.Handler {
	return &Proxy{loadBalancer: loadBalancer}
}

// ServeHTTP implementation of handler for redirect requests to available backend
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var lastErr error
	for t := 0; t < 3; t++ {
		backend, err := p.loadBalancer.Next()
		if err != nil {
			slog.Error("Error get next backend",
				"error", err.Error(),
			)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		director := func(req *http.Request) {
			req.URL.Scheme = backend.GetURL().Scheme
			req.URL.Host = backend.GetURL().Host
			req.Host = backend.GetURL().Host
		}

		tryNext := false
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
			slog.Error("Error in proxy",
				"backend URL", backend.GetURL(),
				"error", e.Error(),
			)
			backend.SetAlive(false)
			lastErr = e
			tryNext = true
		}

		rw := &responseWriterCatcher{ResponseWriter: w}
		proxy.ServeHTTP(rw, r)

		if tryNext {
			continue
		}

		if rw.wroteHeader {
			slog.Info("Proxying to backend",
				"backend URL", backend.GetURL(),
			)
			return
		}
	}

	if lastErr != nil {
		slog.Error("Error in proxy. All attempts failed",
			"error", lastErr.Error(),
		)
	}
	http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
}

// responseWriterCatcher custom ResponseWriter to implement attemptions
type responseWriterCatcher struct {
	http.ResponseWriter
	wroteHeader bool
}

// WriteHeader custom function to implement attemptions
func (r *responseWriterCatcher) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.ResponseWriter.WriteHeader(statusCode)
		r.wroteHeader = true
	}
}
