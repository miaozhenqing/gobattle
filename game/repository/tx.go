package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gobattle/btlog"
)

type txKey struct{}

var errRollbackOnly = errors.New("tx rollback only")

func Transact[T any](ctx context.Context, db *DB, fn func(ctx context.Context, store *TxStore) (T, error)) (res T, err error) {
	// 如果当前处于事务中，则直接执行
	if store, ok := ctx.Value(txKey{}).(*TxStore); ok {
		res, err = fn(ctx, store)
		if err != nil {
			// 内层失败，保证外层也回滚
			store.rollback = true
		}
		return res, err
	}
	// 否则才新建事务
	tx, err := db.sqlxDB.BeginTxx(ctx, nil)
	if err != nil {
		return res, fmt.Errorf("begin tx: %w", err)
	}
	store := newTxStore(tx)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		// 如果外层成功，但内层失败，确保回滚
		if err == nil && store.rollback {
			err = errRollbackOnly
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
			return
		}
		for _, f := range store.afterCommits {
			f()
		}
	}()
	return fn(context.WithValue(ctx, txKey{}, store), store)
}

// 在事务内则最外层提交成功后执行，否则立即执行
func AfterCommit(ctx context.Context, f func()) {
	if store, ok := ctx.Value(txKey{}).(*TxStore); ok {
		store.afterCommits = append(store.afterCommits, f)
		return
	}
	f()
}
