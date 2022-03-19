package calendar

import (
	"fmt"
	"sync"

	"github.com/guilherme-santos/synccalendar"
)

type Mux struct {
	mu       sync.Mutex
	storages map[string]synccalendar.Provider
}

func NewMux() *Mux {
	return &Mux{
		storages: make(map[string]synccalendar.Provider),
	}
}

func (m *Mux) Get(platform string) (synccalendar.Provider, error) {
	storage, ok := m.storages[platform]
	if !ok {
		return nil, fmt.Errorf("calendar %q is not implemented", platform)
	}
	return storage, nil
}

func (m *Mux) Register(platform string, storage synccalendar.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storages[platform] = storage
}
