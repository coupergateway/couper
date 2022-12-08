package plugins

// HookKind describes the plugin mount-points
type HookKind uint8

const (
	Unknown HookKind = iota
	AccessControl
	ProducerRequest
	ProducerProxy
	Connection
)
