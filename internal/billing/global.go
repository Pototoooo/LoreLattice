package billing

import (
	"sync"
)

var (
	defaultMu      sync.RWMutex
	defaultService *Service
)

func SetDefault(service *Service) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultService = service
}

func Default() *Service {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultService
}
