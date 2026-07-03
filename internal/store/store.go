package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
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
`

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil

}
