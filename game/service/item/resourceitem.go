package item

import (
	"context"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
)

type ResourceItemHandler struct{}

func (h *ResourceItemHandler) List(ctx context.Context, store *repository.TxStore, userId int64) ([]*model.ItemBase, error) {
	resources, err := store.Resource.ListByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	bases := make([]*model.ItemBase, 0, len(resources))
	for _, r := range resources {
		bases = append(bases, &r.ItemBase)
	}
	return bases, nil
}

func (h *ResourceItemHandler) UpdateCount(ctx context.Context, store *repository.TxStore, userId int64, deltaByItemId map[int]*StackDelta,
	newId func() int64, now int64) ([]*model.ItemChange, error) {
	itemIds := make([]int, 0, len(deltaByItemId))
	for itemId := range deltaByItemId {
		itemIds = append(itemIds, itemId)
	}
	resources, err := store.Resource.ListByItemIds(ctx, userId, itemIds)
	if err != nil {
		return nil, err
	}
	resourceByItemId := make(map[int]*model.Resource, len(resources))
	for _, res := range resources {
		resourceByItemId[res.ItemId] = res
	}

	upserts := make([]*model.Resource, 0, len(deltaByItemId))
	changes := make([]*model.ItemChange, 0, len(deltaByItemId))
	for itemId, delta := range deltaByItemId {
		res, ok := resourceByItemId[itemId]
		if !ok {
			res = &model.Resource{ItemBase: model.ItemBase{Id: newId(), UserId: userId, ItemId: itemId, CreateTime: now}}
		}
		before := res.Count
		if before < delta.Cost {
			return nil, protocol.CodeItemNotEnough
		}
		after := before - delta.Cost + delta.Add
		if after == before {
			continue
		}
		res.Count = after
		res.UpdateTime = now
		upserts = append(upserts, res)
		changes = append(changes, &model.ItemChange{ItemKey: model.ItemKey{Category: model.ItemCategoryResource, ItemId: itemId},
			Id: res.Id, Before: before, After: after})
	}
	if err = store.Resource.Upsert(ctx, upserts); err != nil {
		return nil, err
	}
	return changes, nil
}
