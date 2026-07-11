package app

import (
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

func LoadConfig() Config {
	root, _ := filepath.Abs(".")
	if filepath.Base(root) == "server" {
		root = filepath.Dir(root)
	}
	dataDir := filepath.Join(root, "data", "server")
	return Config{
		Port:       env("PORT", "8765"),
		Secret:     env("TIME_PLANNER_SECRET", "dev-change-me"),
		AdminToken: env("TIME_PLANNER_ADMIN_TOKEN", ""),
		DataDir:    dataDir,
		DBPath:     filepath.Join(dataDir, "timeplanner.sqlite"),
		DistDir:    filepath.Join(root, "client", "web", "dist"),
		RootDir:    root,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
