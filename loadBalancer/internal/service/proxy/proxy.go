package proxy

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/domain"
	"log/slog"
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	loadBalancer domain.LoadBalancer
}

func NewProxy(loadBalancer domain.LoadBalancer) http.Handler {
	return &Proxy{loadBalancer: loadBalancer}
}

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
			req.URL.Scheme = backend.URL.Scheme
			req.URL.Host = backend.URL.Host
			req.Host = backend.URL.Host
		}

		tryNext := false
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
			slog.Error("Error in proxy",
				"backend URL", backend.URL,
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
				"backend URL", backend.URL,
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

type responseWriterCatcher struct {
	http.ResponseWriter
	wroteHeader bool
}

func (r *responseWriterCatcher) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.ResponseWriter.WriteHeader(statusCode)
		r.wroteHeader = true
	}
}
