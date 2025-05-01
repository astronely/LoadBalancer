package domain

import "github.com/astronely/loadBalancer/loadBalancer/internal/backend"

type LoadBalancer interface {
	Next() (*backend.Backend, error)
}
