package store

import (
	"context"
	"fmt"
	"strings"
)

type Record struct {
	ID         int64  `json:"-"`
	ActorDid   string `json:"did"`
	Collection string `json:"collection"`
	RecordKey  string `json:"rkey"`
}

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

func (s *Store) ListBacklinks(ctx context.Context, target, collection, fieldPath string, dids []string, reverse bool, after int64, limit uint64) (total uint64, records []Record, err error) {
	where := `target = ? AND collection = ? AND field_path = ?`
	args := []any{target, collection, fieldPath}

	if len(dids) > 0 {
		placeholders := make([]string, len(dids))
		for i, did := range dids {
			placeholders[i] = "?"
			args = append(args, did)
		}
		where += `AND actor_did IN (` + strings.Join(placeholders, ", ") + `)`

		err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE `+where, args...).Scan(&total)
		if err != nil {
			return 0, nil, fmt.Errorf("count backlinks: %w", err)
		}
	} else {
		total, err = s.CountBacklinks(ctx, target, collection, fieldPath)
		if err != nil {
			return 0, nil, err
		}
	}

	query := `SELECT id, actor_did, collection, record_key FROM links WHERE ` + where
	listArgs := append([]any{}, args...)

	if after != 0 {
		if reverse {
			query += ` AND id > ?`
		} else {
			query += ` AND id < ?`
		}
		listArgs = append(listArgs, after)
	}

	if reverse {
		query += ` ORDER BY id ASC LIMIT ?`
	} else {
		query += ` ORDER BY id DESC LIMIT ?`
	}
	listArgs = append(listArgs, limit)

	rows, err := s.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return 0, nil, fmt.Errorf("query backlinks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.ActorDid, &rec.Collection, &rec.RecordKey); err != nil {
			return 0, nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterate records: %w", err)
	}

	return total, records, nil
}
