package roundRobin

import (
	"errors"
	"github.com/astronely/loadBalancer/loadBalancer/internal/backend"
	"github.com/astronely/loadBalancer/loadBalancer/internal/domain"
	"sync"
)

type RoundRobin struct {
	backends []*backend.Backend
	mu       sync.RWMutex
	idx      int
}

func NewRoundRobin(b []*backend.Backend) domain.LoadBalancer {
	return &RoundRobin{backends: b}
}

func (r *RoundRobin) Next() (*backend.Backend, error) {
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
