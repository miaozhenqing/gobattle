package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gobattle/btlog"
)

func Transact[T any](ctx context.Context, db *DB, fn func(store *TxStore) (T, error)) (res T, err error) {
	tx, err := db.sqlxDB.BeginTxx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			// ctx 超时后驱动已自动回滚，此时返回 ErrTxDone，忽略
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				btlog.Ins().Error("txRollbackError", "err", rbErr)
			}
			return
		}
		if err = tx.Commit(); err != nil {
			err = fmt.Errorf("commit tx: %w", err)
		}
	}()
	return fn(newTxStore(tx))
}
