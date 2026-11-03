package domain

import "gobattle/game/model"

const (
	ItemReasonGm        = 1
	ItemReasonBattleEnd = 2
	ItemReasonShopBuy   = 3
)

type ItemKey struct {
	Category ItemCategory
	ItemId   int
}

// 增减道具的入参；扣减不可堆叠道具时必须指定 Id（道具表主键），Count 为 1
type ItemAmount struct {
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

type ItemChanges struct {
	UserId  int64
	Reason  int
	Changes []*ItemChange
}

type Item struct {
	Category ItemCategory
	*model.ItemBase
}
