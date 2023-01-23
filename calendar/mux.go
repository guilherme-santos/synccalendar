package calendar

import (
	"fmt"
	"sync"

	"github.com/guilherme-santos/synccalendar"
)

type Mux struct {
	mu        sync.Mutex
	providers map[string]synccalendar.Provider
}

func NewMux() *Mux {
	return &Mux{
		providers: make(map[string]synccalendar.Provider),
	}
}

func (m *Mux) Get(platform string) (synccalendar.Provider, error) {
	storage, ok := m.providers[platform]
	if !ok {
		return nil, fmt.Errorf("calendar %q is not implemented", platform)
	}
	return storage, nil
}

func (m *Mux) Register(platform string, storage synccalendar.Provider) {
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
