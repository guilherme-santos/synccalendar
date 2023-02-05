package calendar

import (
	"fmt"
	"sync"

	"github.com/guilherme-santos/synccalendar/internal"
)

type Mux struct {
	mu        sync.Mutex
	providers map[string]internal.Provider
}

func NewMux() *Mux {
	return &Mux{
		providers: make(map[string]internal.Provider),
	}
}

func (m *Mux) Get(platform string) (internal.Provider, error) {
	storage, ok := m.providers[platform]
	if !ok {
		return nil, fmt.Errorf("calendar %q is not implemented", platform)
	}
	return storage, nil
}

func (m *Mux) Register(platform string, storage internal.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[platform] = storage
}

func (m *Mux) Providers() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	pp := make([]string, 0, len(m.providers))
	for p := range m.providers {
		pp = append(pp, p)
	}
	return pp
}
