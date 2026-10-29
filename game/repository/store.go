package repository

import "github.com/jmoiron/sqlx"

type TxStore struct {
	User *UserRepository
}

func newTxStore(tx *sqlx.Tx) *TxStore {
	return &TxStore{
		User: &UserRepository{tx: tx},
	}
}

type TxSource interface {
	txSource()
}

func (*DB) txSource()      {}
func (*TxStore) txSource() {}
