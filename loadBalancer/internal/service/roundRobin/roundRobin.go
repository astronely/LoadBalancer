package roundRobin

import (
	"errors"
	"github.com/astronely/loadBalancer/loadBalancer/internal/service"
	"sync"
)

// RoundRobin Implementation of loadBalancer algorithm
type RoundRobin struct {
	backends []service.Backend
	mu       sync.RWMutex
	idx      int
}

func NewRoundRobin(b []service.Backend) service.LoadBalancer {
	return &RoundRobin{backends: b}
}

// Next backend to use
func (r *RoundRobin) Next() (service.Backend, error) {
	r.mu.Lock()
	n := len(r.backends)
	r.mu.Unlock()

	if n == 0 {
		return nil, errors.New("no backends found")
	}
	for i := 0; i < n; i++ {
		r.mu.Lock()
		r.idx = (r.idx + 1) % n
		b := r.backends[r.idx]
		r.mu.Unlock()
		if b.IsAlive() {
			return b, nil
		}
	}
	return nil, errors.New("no backends available")
}
