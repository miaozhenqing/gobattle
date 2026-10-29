package model

type ItemBase struct {
	Id         int64 `db:"id"`
	Category   int   `db:"category"`
	Count      int   `db:"count"`
	CreateTime int64 `db:"createTime"`
	UpdateTime int64 `db:"updateTime"`
}

type ItemWeapon struct {
	ItemBase
}
