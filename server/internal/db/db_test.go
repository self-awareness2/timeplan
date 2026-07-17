package db

import (
	"path/filepath"
	"testing"
)

func TestBackupCreatesReadableSnapshot(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.sqlite")
	store, err := Open(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if _, err := store.DB.Exec(`INSERT INTO users (id, username, password_hash, created_at) VALUES ('user-1', 'tester', 'hash', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(t.TempDir(), "backup.sqlite")
	if err := store.Backup(backupPath); err != nil {
		t.Fatal(err)
	}

	backup, err := Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var username string
	if err := backup.DB.QueryRow(`SELECT username FROM users WHERE id = 'user-1'`).Scan(&username); err != nil {
		t.Fatal(err)
	}
	if username != "tester" {
		t.Fatalf("expected backed up user, got %q", username)
	}
}
