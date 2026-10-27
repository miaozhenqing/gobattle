package repository

import "github.com/jmoiron/sqlx"

type DB struct {
	db *sqlx.DB
}

func NewDB(d *sqlx.DB) *DB {
	return &DB{db: d}
}

type Store struct {
	User *UserRepository
}

func newStore(tx *sqlx.Tx) *Store {
	return &Store{
		User: &UserRepository{tx: tx},
	}
}

type Conn interface {
	conn()
}

func (*DB) conn()    {}
func (*Store) conn() {}
