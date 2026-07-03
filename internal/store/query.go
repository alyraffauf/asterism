package store

import (
	"context"
	"fmt"
)

func (s *Store) CountBacklinks(ctx context.Context, target, collection, fieldPath string) (uint64, error) {
	var total uint64

	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM links WHERE target = ? AND collection = ? AND field_path = ?`,
		target, collection, fieldPath,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count backlinks: %w", err)
	}

	return total, nil
}
