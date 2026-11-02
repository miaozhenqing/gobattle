package tpl

type ShopTpl struct {
	Id       int    `json:"id"`
	Type     int    `json:"type"`
	Cost     string `json:"cost"`
	Reward   string `json:"reward"`
	Refresh  int    `json:"refresh"`
	BuyLimit int    `json:"buyLimit"`
}
