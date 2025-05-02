package workerPool

import "net/http"

type Job struct {
	W       http.ResponseWriter
	R       *http.Request
	Handler http.Handler
	Done    chan struct{}
}
