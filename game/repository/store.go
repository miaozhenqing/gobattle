package repository

import "github.com/jmoiron/sqlx"

type TxStore struct {
	User     *UserRepository
	Resource *ResourceRepository
	Weapon   *WeaponRepository
}

func newTxStore(tx *sqlx.Tx) *TxStore {
	return &TxStore{
		User:     &UserRepository{tx: tx},
		Resource: &ResourceRepository{tx: tx},
		Weapon:   &WeaponRepository{tx: tx},
	}
}
