package probe_map

import "sync"

var BackendProbes sync.Map

type HealthInfo struct {
	State   string
	Healthy bool
	Error   string
}
