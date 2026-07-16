package db

import (
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{DB: database}
	if err := store.migrate(); err != nil {
		database.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) migrate() error {
	_, err := s.DB.Exec(`
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS schedules (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	date TEXT NOT NULL,
	start_time TEXT NOT NULL DEFAULT '00:00',
	end_time TEXT NOT NULL DEFAULT '00:00',
	repeat_type TEXT NOT NULL DEFAULT 'none',
	priority TEXT NOT NULL DEFAULT 'medium',
	status TEXT NOT NULL DEFAULT 'pending',
	execution_status TEXT NOT NULL DEFAULT 'not_started',
	actual_start_at TEXT NOT NULL DEFAULT '',
	actual_end_at TEXT NOT NULL DEFAULT '',
	execution_minutes INTEGER NOT NULL DEFAULT 0,
	failure_reason TEXT NOT NULL DEFAULT '',
	category TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_schedules_user_date ON schedules(user_id, date);
CREATE INDEX IF NOT EXISTS idx_schedules_user_status ON schedules(user_id, status);
`)
	if err != nil {
		return err
	}
	for _, statement := range []string{
		`ALTER TABLE schedules ADD COLUMN execution_status TEXT NOT NULL DEFAULT 'not_started'`,
		`ALTER TABLE schedules ADD COLUMN actual_start_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE schedules ADD COLUMN actual_end_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE schedules ADD COLUMN execution_minutes INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE schedules ADD COLUMN failure_reason TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := s.DB.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}
