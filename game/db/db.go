package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gobattle/btlog"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Options struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func Open(ctx context.Context, opts Options) (*sqlx.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = opts.User
	cfg.Passwd = opts.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	cfg.DBName = opts.Database
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	d, err := sqlx.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	d.SetMaxOpenConns(opts.MaxOpenConns)
	d.SetMaxIdleConns(opts.MaxIdleConns)
	d.SetConnMaxLifetime(opts.ConnMaxLifetime)
	d.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	if err = d.PingContext(ctx); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("ping mysql %s: %w", cfg.Addr, err)
	}
	btlog.Ins().Infof("mysql connected, addr:%v db:%v", cfg.Addr, cfg.DBName)
	return d, nil
}

// WithTx fn 返回 error 或 panic 时回滚
func WithTx(ctx context.Context, d *sqlx.DB, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			// ctx 超时后驱动已自动回滚，此时返回 ErrTxDone，忽略
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				btlog.Ins().Errorf("rollback error:%v", rbErr)
			}
			return
		}
		if err = tx.Commit(); err != nil {
			err = fmt.Errorf("commit tx: %w", err)
		}
	}()
	return fn(tx)
}

const (
	errDupEntry = 1062
	errDeadlock = 1213
)

func IsDupEntry(err error) bool {
	return mysqlErrNo(err) == errDupEntry
}

func IsDeadlock(err error) bool {
	return mysqlErrNo(err) == errDeadlock
}

func mysqlErrNo(err error) uint16 {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number
	}
	return 0
}
