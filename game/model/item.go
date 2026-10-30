package model

const (
	ItemCategoryGold = 1 // 金币
)

// 道具持有方式
const (
	ItemStackMerge  = 1 // 每道具一行累加，instanceId 为 0
	ItemStackBatch  = 2 // 每次获得一行，带过期时间
	ItemStackUnique = 3 // 一件一行，带专属数据
)

const (
	ItemReasonGm        = 1
	ItemReasonBattleEnd = 2
)

type Item struct {
	Id       int64 `db:"id"`
	UserId   int64 `db:"userId"`
	Category int   `db:"category"`
	ItemId   int   `db:"itemId"`
	//把道具统一存一个表，这个字段用于处理不可堆叠的道具
	InstanceId int64 `db:"instanceId"`
	Count      int64 `db:"count"`
	ExpireTime int64 `db:"expireTime"`
	// 道具专属数据
	Data       []byte `db:"data"`
	CreateTime int64  `db:"createTime"`
	UpdateTime int64  `db:"updateTime"`
}

func (i *Item) Key() ItemKey {
	return ItemKey{Category: i.Category, ItemId: i.ItemId}
}

type ItemKey struct {
	Category int
	ItemId   int
}

// 增减道具的入参
type ItemChangeReq struct {
	ItemKey
	Count int64
}

// 道具变动
type ItemChange struct {
	ItemKey
	Before int64
	After  int64
}

type ItemChangeResult struct {
	UserId  int64
	Reason  int
	Changes []*ItemChange
}
