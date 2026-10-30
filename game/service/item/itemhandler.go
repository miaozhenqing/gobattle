package item

import (
	"context"
	"gobattle/game/model"
	"gobattle/game/repository"
)

type Handler interface {
	List(ctx context.Context, store *repository.TxStore, userId int64) ([]*model.ItemBase, error)
}

// 可堆叠道具
type StackHandler interface {
	Handler
	UpdateCount(ctx context.Context, store *repository.TxStore, userId int64, deltaByItemId map[int]*StackDelta,
		newId func() int64, now int64) ([]*model.ItemChange, error)
}

// 不可堆叠道具
type UnstackHandler interface {
	Handler
	Add(ctx context.Context, store *repository.TxStore, userId int64, itemId int, ids []int64, now int64) error

	Delete(ctx context.Context, store *repository.TxStore, userId int64, itemIdById map[int64]int) error
}

var stackHandlerByCategory = map[model.ItemCategory]StackHandler{
	model.ItemCategoryResource: &ResourceItemHandler{},
}

var unstackHandlerByCategory = map[model.ItemCategory]UnstackHandler{
	model.ItemCategoryWeapon: &weaponItemHandler{},
}

func GetHandler(category model.ItemCategory) Handler {
	if h, ok := stackHandlerByCategory[category]; ok {
		return h
	}
	return unstackHandlerByCategory[category]
}

func GetStackHandler(category model.ItemCategory) (StackHandler, bool) {
	h, ok := stackHandlerByCategory[category]
	return h, ok
}

func GetUnstackHandler(category model.ItemCategory) (UnstackHandler, bool) {
	h, ok := unstackHandlerByCategory[category]
	return h, ok
}

type StackDelta struct {
	Cost int64
	Add  int64
}
