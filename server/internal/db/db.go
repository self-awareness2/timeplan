package db

import (
	"database/sql"
	"fmt"
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
	if _, err := s.DB.Exec(`PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);`); err != nil {
		return err
	}

	migrations := []migration{
		{version: 1, apply: migrateInitialSchema},
		{version: 2, apply: migrateExecutionFields},
		{version: 3, apply: migrateRecurringOccurrences},
	}
	for _, migration := range migrations {
		var applied bool
		if err := s.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, migration.version).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := migration.apply(s.DB); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err := s.DB.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))`, migration.version); err != nil {
			return err
		}
	}
	return nil
}

type migration struct {
	version int
	apply   func(*sql.DB) error
}

func migrateInitialSchema(database *sql.DB) error {
	_, err := database.Exec(`
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
	series_parent_id INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_schedules_user_date ON schedules(user_id, date);
CREATE INDEX IF NOT EXISTS idx_schedules_user_status ON schedules(user_id, status);
`)
	return err
}

func migrateExecutionFields(database *sql.DB) error {
	for _, statement := range []string{
		`ALTER TABLE schedules ADD COLUMN execution_status TEXT NOT NULL DEFAULT 'not_started'`,
		`ALTER TABLE schedules ADD COLUMN actual_start_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE schedules ADD COLUMN actual_end_at TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE schedules ADD COLUMN execution_minutes INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE schedules ADD COLUMN failure_reason TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := database.Exec(statement); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}

func migrateRecurringOccurrences(database *sql.DB) error {
	if _, err := database.Exec(`ALTER TABLE schedules ADD COLUMN series_parent_id INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	_, err := database.Exec(`
CREATE TABLE IF NOT EXISTS schedule_exceptions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id TEXT NOT NULL,
	schedule_id INTEGER NOT NULL,
	occurrence_date TEXT NOT NULL,
	created_at TEXT NOT NULL,
	UNIQUE(user_id, schedule_id, occurrence_date),
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_schedules_user_series ON schedules(user_id, series_parent_id);
CREATE INDEX IF NOT EXISTS idx_schedule_exceptions_user_date ON schedule_exceptions(user_id, occurrence_date);
`)
	return err
}

func (s *Store) Backup(path string) error {
	escapedPath := strings.ReplaceAll(path, "'", "''")
	_, err := s.DB.Exec(`VACUUM INTO '` + escapedPath + `'`)
	return err
}
