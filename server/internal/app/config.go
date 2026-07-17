package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port            string
	Environment     string
	Secret          string
	AdminToken      string
	AuthMaxAttempts int
	DataDir         string
	BackupDir       string
	DBPath          string
	DistDir         string
	RootDir         string
	AllowedOrigins  []string
}

func (c Config) Validate() error {
	if c.Environment == "production" && (c.Secret == "" || c.Secret == "dev-change-me") {
		return fmt.Errorf("CHRONA_SECRET must be set to a non-default value")
	}
	return nil
}

func LoadConfig() Config {
	root := projectRoot()
	dataDir := env("CHRONA_DATA_DIR", filepath.Join(root, "data", "server"))
	return Config{
		Port:            env("PORT", "8765"),
		Environment:     env("CHRONA_ENV", "development"),
		Secret:          env("CHRONA_SECRET", "dev-change-me"),
		AdminToken:      env("CHRONA_ADMIN_TOKEN", ""),
		AuthMaxAttempts: envInt("CHRONA_AUTH_MAX_ATTEMPTS", 10),
		DataDir:         dataDir,
		BackupDir:       env("CHRONA_BACKUP_DIR", filepath.Join(dataDir, "backups")),
		DBPath:          filepath.Join(dataDir, "chrona.sqlite"),
		DistDir:         env("CHRONA_DIST_DIR", filepath.Join(root, "client", "web", "dist")),
		RootDir:         root,
		AllowedOrigins:  csvEnv("CHRONA_ALLOWED_ORIGINS", []string{"http://localhost", "https://localhost", "capacitor://localhost"}),
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

func csvEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	values := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if origin := strings.TrimSpace(item); origin != "" {
			values = append(values, origin)
		}
	}
	return values
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
