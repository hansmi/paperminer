package store

import (
	"context"
	"fmt"
	"time"

	"github.com/timshannon/bolthold"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

func Prune(ctx context.Context, logger *zap.Logger, s *bolthold.Store, deleteUpdatedBefore time.Time) error {
	query := bolthold.Where("RecordUpdated").Lt(deleteUpdatedBefore)

	return s.Bolt().Update(func(tx *bbolt.Tx) error {
		var dataType DocumentTask

		count, err := s.TxCount(tx, dataType, query)
		if err != nil {
			return fmt.Errorf("counting tasks: %w", err)
		}
		if count == 0 {
			return nil
		}

		logger.Debug("Delete obsolete records from store",
			zap.Int("count", count),
			zap.Time("updated_before", deleteUpdatedBefore),
		)

		if err := s.TxDeleteMatching(tx, dataType, query); err != nil {
			return fmt.Errorf("deleting records: %w", err)
		}

		return nil
	})
}
