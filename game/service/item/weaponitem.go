package item

import (
	"context"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
)

type weaponItemHandler struct{}

func (h *weaponItemHandler) List(ctx context.Context, store *repository.TxStore, userId int64) ([]*model.ItemBase, error) {
	weapons, err := store.Weapon.ListByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	bases := make([]*model.ItemBase, 0, len(weapons))
	for _, w := range weapons {
		bases = append(bases, &w.ItemBase)
	}
	return bases, nil
}

func (h *weaponItemHandler) Add(ctx context.Context, store *repository.TxStore, userId int64, itemId int, ids []int64, now int64) error {
	weapons := make([]*model.Weapon, 0, len(ids))
	for _, id := range ids {
		weapons = append(weapons, &model.Weapon{ItemBase: model.ItemBase{Id: id, UserId: userId, ItemId: itemId, Count: 1, CreateTime: now, UpdateTime: now},
			Level: 1})
	}
	return store.Weapon.Insert(ctx, weapons)
}

func (h *weaponItemHandler) Delete(ctx context.Context, store *repository.TxStore, userId int64, itemIdById map[int64]int) error {
	ids := make([]int64, 0, len(itemIdById))
	for id := range itemIdById {
		ids = append(ids, id)
	}
	weapons, err := store.Weapon.ListByIds(ctx, userId, ids)
	if err != nil {
		return err
	}
	if len(weapons) != len(ids) {
		return protocol.CodeItemNotEnough
	}
	for _, w := range weapons {
		if itemIdById[w.Id] != w.ItemId {
			return protocol.CodeItemNotEnough
		}
	}
	return store.Weapon.DeleteByIds(ctx, userId, ids)
}
