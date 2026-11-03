package model

// 道具公共字段，各类型道具表都包含这些列
type ItemBase struct {
	Id     int64 `db:"id"`
	UserId int64 `db:"userId"`
	ItemId int   `db:"itemId"`
	// 不可堆叠道具固定为 1
	Count      int64 `db:"count"`
	CreateTime int64 `db:"createTime"`
	UpdateTime int64 `db:"updateTime"`
}

// 资源
type UserResource struct {
	ItemBase
}

// 武器
type UserWeapon struct {
	ItemBase
	Level int `db:"level"`
}
