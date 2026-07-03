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

type ManyToManyItem struct {
	LinkRecord   Record `json:"linkRecord"`
	OtherSubject string `json:"otherSubject"`
}


type OtherSubjectCount struct {
	Subject  string `json:"subject"`
	Total    uint64 `json:"total"`
	Distinct uint64 `json:"distinct"`
}

func (s *Store) CountBacklinks(ctx context.Context, target, collection, fieldPath string) (uint64, error) {
	var total uint64

	err := s.readDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM links WHERE target = ? AND collection = ? AND field_path = ?`,
		target, collection, fieldPath,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count backlinks: %w", err)
	}

	return total, nil
}

func (s *Store) DistinctBacklinkDids(ctx context.Context, target, collection, fieldPath string, after string, limit uint64) (total uint64, dids []string, err error) {
	err = s.readDB.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT actor_did) FROM links WHERE target = ? AND collection = ? AND field_path = ?`,
		target, collection, fieldPath,
	).Scan(&total)
	if err != nil {
		return 0, nil, fmt.Errorf("count distinct dids: %w", err)
	}

	rows, err := s.readDB.QueryContext(ctx,
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

		err = s.readDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM links WHERE `+where, args...).Scan(&total)
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

	rows, err := s.readDB.QueryContext(ctx, query, listArgs...)
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

func (s *Store) ManyToMany(ctx context.Context, target, collection, fieldPath, pathToOther string, linkDids, otherSubjects []string, after int64, limit uint64) (total uint64, items []ManyToManyItem, err error) {
	where := `a.target = ? AND a.collection = ? AND a.field_path = ? AND b.field_path = ?`
	args := []any{target, collection, fieldPath, pathToOther}

	if len(linkDids) > 0 {
		placeholders := make([]string, len(linkDids))
		for i, did := range linkDids {
			placeholders[i] = "?"
			args = append(args, did)
		}
		where += ` AND a.actor_did IN (` + strings.Join(placeholders, ", ") + `)`
	}

	if len(otherSubjects) > 0 {
		placeholders := make([]string, len(otherSubjects))
		for i, subj := range otherSubjects {
			placeholders[i] = "?"
			args = append(args, subj)
		}
		where += ` AND b.target IN (` + strings.Join(placeholders, ", ") + `)`
	}

	joinClause := ` FROM links a JOIN links b
		ON a.actor_did = b.actor_did AND a.collection = b.collection AND a.record_key = b.record_key
		WHERE ` + where

	if err := s.readDB.QueryRowContext(ctx, `SELECT COUNT(*)`+joinClause, args...).Scan(&total); err != nil {
		return 0, nil, fmt.Errorf("count many to many: %w", err)
	}

	listQuery := `SELECT a.id, a.actor_did, a.collection, a.record_key, b.target` + joinClause
	listArgs := append([]any{}, args...)

	if after != 0 {
		listQuery += ` AND a.id > ?`
		listArgs = append(listArgs, after)
	}

	listQuery += ` ORDER BY a.id ASC LIMIT ?`
	listArgs = append(listArgs, limit)

	rows, err := s.readDB.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return 0, nil, fmt.Errorf("query many to many: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item ManyToManyItem
		if err := rows.Scan(&item.LinkRecord.ID, &item.LinkRecord.ActorDid, &item.LinkRecord.Collection, &item.LinkRecord.RecordKey, &item.OtherSubject); err != nil {
			return 0, nil, fmt.Errorf("scan many to many item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, fmt.Errorf("iterate many to many: %w", err)
	}

	return total, items, nil
}

func (s *Store) ManyToManyCounts(ctx context.Context, target, collection, fieldPath, pathToOther string, linkDids, otherSubjects []string, after string, limit uint64) (counts []OtherSubjectCount, err error) {
	where := `a.target = ? AND a.collection = ? AND a.field_path = ? AND b.field_path = ?`
	args := []any{target, collection, fieldPath, pathToOther}

	if len(linkDids) > 0 {
		placeholders := make([]string, len(linkDids))
		for i, did := range linkDids {
			placeholders[i] = "?"
			args = append(args, did)
		}
		where += ` AND a.actor_did IN (` + strings.Join(placeholders, ", ") + `)`
	}

	if len(otherSubjects) > 0 {
		placeholders := make([]string, len(otherSubjects))
		for i, subj := range otherSubjects {
			placeholders[i] = "?"
			args = append(args, subj)
		}
		where += ` AND b.target IN (` + strings.Join(placeholders, ", ") + `)`
	}

	where += ` AND b.target > ?`
	args = append(args, after)

	query := `SELECT b.target, COUNT(*), COUNT(DISTINCT a.actor_did)
		FROM links a JOIN links b
		ON a.actor_did = b.actor_did AND a.collection = b.collection AND a.record_key = b.record_key
		WHERE ` + where + `
		GROUP BY b.target
		ORDER BY b.target
		LIMIT ?`
	args = append(args, limit)

	rows, err := s.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query many to many counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c OtherSubjectCount
		if err := rows.Scan(&c.Subject, &c.Total, &c.Distinct); err != nil {
			return nil, fmt.Errorf("scan many to many count: %w", err)
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate many to many counts: %w", err)
	}

	return counts, nil
}
