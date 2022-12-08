package connection

type Configuration struct {
	// ContextName maps as value to ContextType
	ContextName string
	// ContextType appears as connection metrics and log label
	ContextType string
	// Hostname may differ for this Origin address
	Hostname string
	// Origin dial address
	Origin string
}
