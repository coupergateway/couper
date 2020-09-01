package config

type CORS struct {
	AllowedOrigins   []string `hcl:"allowed_origins"`	// TODO auch string erlauben
	AllowCredentials bool     `hcl:"allow_credentials,optional"`
	MaxAge           string   `hcl:"max_age,optional"`
}
