package repository

import "github.com/jmoiron/sqlx"

type TxStore struct {
	User *UserRepository
	Item *ItemRepository
}

func newTxStore(tx *sqlx.Tx) *TxStore {
	return &TxStore{
		User: &UserRepository{tx: tx},
		Item: &ItemRepository{tx: tx},
	}
}
