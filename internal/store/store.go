package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	writeDB *sql.DB
	readDB  *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS links (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	target      TEXT NOT NULL,
	target_cid  TEXT NOT NULL,
	actor_did   TEXT NOT NULL,
	collection  TEXT NOT NULL,
	record_key  TEXT NOT NULL,
	record_cid  TEXT NOT NULL,
	field_path  TEXT NOT NULL,
	rev         TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_links_target ON links(target);
CREATE INDEX IF NOT EXISTS idx_links_source ON links(actor_did, collection, record_key);
CREATE TABLE IF NOT EXISTS cursor (
	id  INTEGER PRIMARY KEY CHECK (id = 0),
	seq INTEGER NOT NULL
);
`

func Open(path string) (*Store, error) {
	writeDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open write database: %w", err)
	}

	if _, err := writeDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := writeDB.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	if _, err := writeDB.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	readDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open read database: %w", err)
	}
	if _, err := readDB.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("set read busy timeout: %w", err)
	}

	return &Store{writeDB: writeDB, readDB: readDB}, nil
}
