package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"chrona/server/internal/app"
	"chrona/server/internal/db"
)

func main() {
	cfg := app.LoadConfig()
	if err := cfg.Validate(); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(cfg.BackupDir, 0o755); err != nil {
		fatal(err)
	}

	store, err := db.Open(cfg.DBPath)
	if err != nil {
		fatal(err)
	}
	defer store.Close()

	path := filepath.Join(cfg.BackupDir, "chrona-"+time.Now().UTC().Format("20060102-150405")+".sqlite")
	if err := store.Backup(path); err != nil {
		fatal(err)
	}
	fmt.Println(path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "backup failed:", err)
	os.Exit(1)
}
