package tpl

import "gobattle/game/domain"

type ItemTpl struct {
	Id       int                 `json:"id"`
	Category domain.ItemCategory `json:"category"`
	Name     string              `json:"name"`
}
