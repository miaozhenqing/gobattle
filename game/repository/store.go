package repository

import "github.com/jmoiron/sqlx"

type TxStore struct {
	UserRepository     *UserRepository
	ResourceRepository *ResourceRepository
	WeaponRepository   *WeaponRepository
	ShopRepository     *ShopRepository

	rollback     bool
	afterCommits []func()
}

func newTxStore(tx *sqlx.Tx) *TxStore {
	return &TxStore{
		UserRepository:     &UserRepository{tx: tx},
		ResourceRepository: &ResourceRepository{tx: tx},
		WeaponRepository:   &WeaponRepository{tx: tx},
		ShopRepository:     &ShopRepository{tx: tx},
	}
}
