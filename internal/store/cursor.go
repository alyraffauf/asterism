package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) SaveCursor(ctx context.Context, seq int64) error {
	_, err := s.writeDB.ExecContext(ctx,
		`INSERT INTO cursor (id, seq) VALUES (0, ?) ON CONFLICT (id) DO UPDATE SET seq = excluded.seq`, seq,
	)

	if err != nil {
		return fmt.Errorf("save cursor: %w", err)
	}
	return nil
}


func (s *Store) GetCursor(ctx context.Context) (int64, error) {
	var seq int64
	err := s.readDB.QueryRowContext(ctx, `SELECT seq FROM cursor WHERE id = 0`).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("get cursor: %w", err)
	}

	return seq, nil
}
