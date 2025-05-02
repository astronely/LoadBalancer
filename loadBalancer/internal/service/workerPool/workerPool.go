package workerPool

import (
	"github.com/astronely/loadBalancer/loadBalancer/internal/config"
	"github.com/astronely/loadBalancer/loadBalancer/internal/model/workerPool"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"log/slog"
)

type WorkerPool struct {
	poolSize  int
	queueSize int
}

func NewWorkerPool(cfg config.WorkerPoolConfig) service.WorkerPool {
	return &WorkerPool{
		poolSize:  cfg.PoolSize(),
		queueSize: cfg.QueueSize(),
	}
}

func (w *WorkerPool) Start() chan workerPool.Job {
	jobs := make(chan workerPool.Job, w.queueSize)
	for i := 0; i < w.poolSize; i++ {
		slog.Debug("Worker Pool",
			"id", i,
			"job capacity", cap(jobs),
		)
		go w.Worker(jobs)
	}
	return jobs
}

func (w *WorkerPool) Worker(jobs <-chan workerPool.Job) {
	for job := range jobs {
		job.Handler.ServeHTTP(job.W, job.R)
		job.Done <- struct{}{}
	}
}
