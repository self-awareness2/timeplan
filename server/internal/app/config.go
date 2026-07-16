package app

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port       string
	Secret     string
	AdminToken string
	DataDir    string
	DBPath     string
	DistDir    string
	RootDir    string
}

func (c Config) Validate() error {
	if os.Getenv("CHRONA_ENV") == "production" && (c.Secret == "" || c.Secret == "dev-change-me") {
		return fmt.Errorf("CHRONA_SECRET must be set to a non-default value")
	}
	return nil
}

func LoadConfig() Config {
	root := projectRoot()
	dataDir := filepath.Join(root, "data", "server")
	return Config{
		Port:       env("PORT", "8765"),
		Secret:     env("CHRONA_SECRET", "dev-change-me"),
		AdminToken: env("CHRONA_ADMIN_TOKEN", ""),
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "chrona.sqlite"),
		DistDir:    filepath.Join(root, "client", "web", "dist"),
		RootDir:    root,
	}
}

func projectRoot() string {
	if configured := os.Getenv("CHRONA_ROOT"); configured != "" {
		if absolute, err := filepath.Abs(configured); err == nil {
			return absolute
		}
	}
	start, _ := filepath.Abs(".")
	for current := start; current != filepath.Dir(current); current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "client", "web", "dist", "index.html")); err == nil {
			return current
		}
		if filepath.Base(current) == "server" {
			return filepath.Dir(current)
		}
	}
	return start
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
