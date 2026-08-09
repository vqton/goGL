package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Server        Server        `yaml:"server"`
	Database      Database      `yaml:"database"`
	Authorization Authorization `yaml:"authorization"`
}

type Server struct {
	HTTPAddr string `yaml:"http_addr"`
}

type Database struct {
	DSN string `yaml:"dsn"`
}

// Authorization configures casbin-based access control on the /api/v1 group.
type Authorization struct {
	// Enabled toggles the enforcement middleware. Leave on; it is the
	// authorization boundary for the whole API.
	Enabled bool `yaml:"enabled"`
	// IdentityHeader names the header the principal resolver reads the
	// subject (user id) from. Dev seam until real authentication lands.
	IdentityHeader string `yaml:"identity_header"`
}

func Load(path string) *Config {
	cfg := &Config{
		Server:   Server{HTTPAddr: ":8080"},
		Database: Database{DSN: "gogl.db"},
		Authorization: Authorization{
			Enabled:        true,
			IdentityHeader: "X-User-Id",
		},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(data, cfg)
	return cfg
}
