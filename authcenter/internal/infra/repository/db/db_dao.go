package db

import (
	"context"
	"fmt"

	"github.com/RoyceAzure/lab/authcenter/internal/infra/repository/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IStore interface {
	sqlc.Querier
	ExecTx(ctx context.Context, fn func(*sqlc.Queries) error) error
	ExecMultiTx(ctx context.Context, fns []func(*sqlc.Queries) error) error
}

// Store 結構用來管理數據庫連接和交易
type Store struct {
	*sqlc.Queries
	db *pgxpool.Pool
}

// NewStore 創建一個新的 Store
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: sqlc.New(db),
	}
}

// ExecTx 執行一個交易
func (s *Store) ExecTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	opts := pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted, // 最常用的隔離級別
		AccessMode:     pgx.ReadWrite,     // 需要寫入時使用
		DeferrableMode: pgx.NotDeferrable, // 通常使用立即檢查
	}

	tx, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	q := sqlc.New(tx)
	err = fn(q)

	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit(ctx)
}

// ExecMultiTx 執行多個tx
func (s *Store) ExecMultiTx(ctx context.Context, fns []func(*sqlc.Queries) error) error {
	opts := pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted, // 最常用的隔離級別
		AccessMode:     pgx.ReadWrite,     // 需要寫入時使用
		DeferrableMode: pgx.NotDeferrable, // 通常使用立即檢查
	}

	tx, err := s.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	q := sqlc.New(tx)

	for _, fn := range fns {
		err = fn(q)
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return fmt.Errorf("原始錯誤: %v, 回滾錯誤: %v", err, rbErr)
			}
			return err
		}
	}

	return tx.Commit(ctx)
}
