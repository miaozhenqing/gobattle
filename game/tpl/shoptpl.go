package tpl

const (
	RefreshNone    = 0
	RefreshDaily   = 1
	RefreshWeekly  = 2
	RefreshMonthly = 3
)

type ShopTpl struct {
	Id       int    `json:"id"`
	Type     int    `json:"type"`
	Cost     string `json:"cost"`
	Reward   string `json:"reward"`
	Refresh  int    `json:"refresh"`
	BuyLimit int    `json:"buyLimit"`
}
