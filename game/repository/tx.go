package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gobattle/btlog"
)

// Tx 读写事务。c 为 *DB 时新开事务，为 *Store 时加入当前事务。
// fn 返回 error 或 panic 时回滚；fn 里只做数据库操作，绑 session、推消息放到事务外
func Tx[T any](ctx context.Context, src TxSource, fn func(store *TxStore) (T, error)) (res T, err error) {
	switch v := src.(type) {
	case *TxStore:
		return fn(v)
	case *DB:
		return newTx(ctx, v, fn)
	default:
		return res, fmt.Errorf("unknown tx source type %T", src)
	}
}

func newTx[T any](ctx context.Context, d *DB, fn func(store *TxStore) (T, error)) (res T, err error) {
	tx, err := d.sqlxDB.BeginTxx(ctx, nil)
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
