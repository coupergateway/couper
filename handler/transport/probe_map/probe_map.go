package probe_map

import (
	"sync"
)

type BackendProbes map[string]string

var (
	probeMutex       sync.RWMutex
	BackendProbesMap = make(BackendProbes)
)

func SetBackendProbe(name string, state string) {
	probeMutex.Lock()
	BackendProbesMap[name] = state
	probeMutex.Unlock()
}

func GetBackendProbes() BackendProbes {
	cp := make(BackendProbes)

	probeMutex.RLock()
	for name, state := range BackendProbesMap {
		cp[name] = state
	}
	probeMutex.RUnlock()

	return cp
}
