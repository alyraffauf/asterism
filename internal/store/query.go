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

func (s *Store) DistinctBacklinkDids(ctx context.Context, target, collection, fieldPath string, after string, limit uint64) (total uint64, dids []string, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT actor_did) FROM links WHERE target = ? AND collection = ? AND field_path = ?`,
		target, collection, fieldPath,
	).Scan(&total)
	if err != nil {
		return 0, nil, fmt.Errorf("count distinct dids: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT actor_did FROM links
		 WHERE target = ? AND collection = ? AND field_path = ? AND actor_did > ?
		 ORDER BY actor_did LIMIT ?`,
		target, collection, fieldPath, after, limit,
	)
	if err != nil {
		return 0, nil, fmt.Errorf("query distinct dids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			return 0, nil, fmt.Errorf("scan did: %w", err)
		}
		dids = append(dids, did)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterate dids: %w", err)
	}

	return total, dids, nil
}
