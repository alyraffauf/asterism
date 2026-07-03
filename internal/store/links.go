package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alyraffauf/asterism/internal/backlink"
)

func deleteLinks(ctx context.Context, tx *sql.Tx, actorDid, collection, recordKey string) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM links WHERE actor_did = ? AND collection = ? AND record_key = ?`,
		actorDid, collection, recordKey,
	)

	if err != nil {
		return fmt.Errorf("delete links: %w", err)
	}
	return nil
}

func (s *Store) DeleteLinks(ctx context.Context, actorDid, collection, recordKey string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := deleteLinks(ctx, tx, actorDid, collection, recordKey); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) SaveLinks(ctx context.Context, actorDid, collection, recordKey string, links []backlink.Link) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := deleteLinks(ctx, tx, actorDid, collection, recordKey); err != nil {
		return err
	}

	for _, link := range links {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO links (target, target_cid, actor_did, collection, record_key, record_cid, field_path, rev)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			link.Target, link.TargetCid, link.ActorDid, link.Collection, link.RecordKey, link.RecordCid, link.FieldPath, link.Rev,
		)
		if err != nil {
			return fmt.Errorf("insert link: %w", err)
		}
	}

	return tx.Commit()
}
