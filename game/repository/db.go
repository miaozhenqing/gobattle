package repository

import (
	"context"
	"errors"
	"fmt"
	"gobattle/btlog"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DBOptions struct {
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

type DB struct {
	sqlxDB *sqlx.DB
}

func OpenDB(ctx context.Context, opts DBOptions) (*DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = opts.User
	cfg.Passwd = opts.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	cfg.DBName = opts.Database
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	sqlxDB, err := sqlx.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlxDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlxDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlxDB.SetConnMaxLifetime(opts.ConnMaxLifetime)
	sqlxDB.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	if err = sqlxDB.PingContext(ctx); err != nil {
		_ = sqlxDB.Close()
		return nil, fmt.Errorf("ping mysql %s: %w", cfg.Addr, err)
	}
	btlog.Ins().Info("mysqlConnected", "addr", cfg.Addr, "db", cfg.DBName)
	return &DB{sqlxDB: sqlxDB}, nil
}

func (d *DB) Close() error {
	return d.sqlxDB.Close()
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
