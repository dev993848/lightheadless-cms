package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	Port       int
	DBPath     string
	UploadPath string
}

// Load parses CLI flags and environment variables.
// Environment variables override flags.
// Validates that Port is in range 1-65535.
func Load() (*Config, error) {
	var (
		flagPort       int
		flagDBPath     string
		flagUploadPath string
	)

	flag.IntVar(&flagPort, "port", 8080, "HTTP listen port")
	flag.StringVar(&flagDBPath, "db", "./cms.db", "SQLite database path")
	flag.StringVar(&flagUploadPath, "upload", "./uploads", "Upload directory path")
	flag.Parse()

	cfg := &Config{
		Port:       flagPort,
		DBPath:     flagDBPath,
		UploadPath: flagUploadPath,
	}

	// Environment variables override flags.
	if v := os.Getenv("CMS_PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid CMS_PORT value %q: %w", v, err)
		}
		cfg.Port = p
	}
	if v := os.Getenv("CMS_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("CMS_UPLOAD_PATH"); v != "" {
		cfg.UploadPath = v
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("port %d is out of valid range 1-65535", cfg.Port)
	}

	return cfg, nil
}
