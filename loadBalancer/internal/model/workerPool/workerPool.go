package workerPool

import "net/http"

// Job model to use in WorkPool
type Job struct {
	W       http.ResponseWriter
	R       *http.Request
	Handler http.Handler
	Done    chan struct{}
}
