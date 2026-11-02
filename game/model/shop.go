package model

type UserShop struct {
	Id         int64 `db:"id"`
	UserId     int64 `db:"userId"`
	CreateTime int64 `db:"createTime"`
	UpdateTime int64 `db:"updateTime"`

	ShopId      int   `db:"shopId"`
	BuyCount    int   `db:"buyCount"`
	LastBuyTime int64 `db:"lastBuyTime"`
}
