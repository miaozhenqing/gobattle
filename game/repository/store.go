package repository

import "github.com/jmoiron/sqlx"

type TxStore struct {
	User     *UserRepository
	Resource *ResourceRepository
	Weapon   *WeaponRepository
	Shop     *ShopRepository

	rollback     bool
	afterCommits []func()
}

func newTxStore(tx *sqlx.Tx) *TxStore {
	return &TxStore{
		User:     &UserRepository{tx: tx},
		Resource: &ResourceRepository{tx: tx},
		Weapon:   &WeaponRepository{tx: tx},
		Shop:     &ShopRepository{tx: tx},
	}
}
