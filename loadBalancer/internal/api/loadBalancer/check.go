package loadBalancer

import (
	"context"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/workerPool"
	"net/http"
	"time"
)

// Check handler using WorkPool pattern
func (i *Implementation) Check(ctx context.Context) {
	jobs := i.workerPool.Start()

	i.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Second*1)

		jobDone := make(chan struct{})
		select {
		case jobs <- workerPool.Job{W: w, R: r, Handler: i.proxy, Done: jobDone}:
			<-jobDone
			cancel()
		case <-ctxWithTimeout.Done():
			http.Error(w, "Timeout, servers unavailable", http.StatusServiceUnavailable)
			cancel()
		}
	})
}
