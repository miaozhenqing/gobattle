package model

const (
	ItemReasonGm        = 1
	ItemReasonBattleEnd = 2
)

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
type Resource struct {
	ItemBase
}

// 武器
type Weapon struct {
	ItemBase
	Level int `db:"level"`
}

// itemId 类型内唯一
type ItemKey struct {
	Category ItemCategory
	ItemId   int
}

// 增减道具的入参；扣减不可堆叠道具时必须指定 Id（道具表主键），Count 为 1
type ItemChangeReq struct {
	ItemKey
	Id    int64
	Count int64
}

// 道具变动；不可堆叠道具带 Id
type ItemChange struct {
	ItemKey
	Id     int64
	Before int64
	After  int64
}

type ItemChangeResult struct {
	UserId  int64
	Reason  int
	Changes []*ItemChange
}
