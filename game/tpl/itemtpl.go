package tpl

import "gobattle/game/model"

type ItemTpl struct {
	Id       int                `json:"id"`
	Category model.ItemCategory `json:"category"`
	Name     string             `json:"name"`
}
