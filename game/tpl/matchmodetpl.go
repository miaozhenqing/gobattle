package tpl

import (
	"slices"
	"time"
)

type MatchModeTpl struct {
	Id              int    `json:"id"`
	Name            string `json:"name"`
	PlayerCount     int    `json:"playerCount"`
	MapIds          []int  `json:"mapIds"`
	MatchTimeoutSec int    `json:"matchTimeoutSec"`
}

func (t *MatchModeTpl) HasMap(mapId int) bool {
	return slices.Contains(t.MapIds, mapId)
}

func (t *MatchModeTpl) MatchTimeout() time.Duration {
	return time.Duration(t.MatchTimeoutSec) * time.Second
}
