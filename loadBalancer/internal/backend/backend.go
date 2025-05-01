package backend

import (
	"log/slog"
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL   *url.URL
	alive uint32
}

func NewBackend(address string) (*Backend, error) {
	backendUrl, err := url.Parse(address)
	if err != nil {
		slog.Error("error parse backendUrl",
			"error", err.Error(),
		)
		return nil, err
	}
	b := &Backend{URL: backendUrl}
	atomic.StoreUint32(&b.alive, 1)
	return b, nil
}

func (b *Backend) SetAlive(value bool) {
	var val uint32
	if value {
		val = 1
	}
	atomic.StoreUint32(&b.alive, val)
}

func (b *Backend) IsAlive() bool {
	return atomic.LoadUint32(&b.alive) == 1
}
