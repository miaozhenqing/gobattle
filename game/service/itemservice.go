package service

import (
	"context"
	"errors"
	"gobattle/btlog"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"time"
)

var itemStackByCategory = map[int]int{
	model.ItemCategoryGold: model.ItemStackMerge,
}

// 道具业务
type ItemService struct {
	db           *repository.DB
	pushService  *PushService
	queryTimeout time.Duration
}

func NewItemService(db *repository.DB, pushService *PushService, queryTimeout time.Duration) *ItemService {
	return &ItemService{db: db, pushService: pushService, queryTimeout: queryTimeout}
}

// 仓库道具列表
func (s *ItemService) List(ctx context.Context, userId int64) ([]*model.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	items, err := repository.Transact(ctx, s.db, func(store *repository.TxStore) ([]*model.Item, error) {
		return store.Item.ListByUserId(ctx, userId, time.Now().UnixMilli())
	})
	if err != nil {
		return nil, dbFailed(err, "list item userId:%d", userId)
	}
	return items, nil
}

// 批量查数量
func (s *ItemService) GetCounts(ctx context.Context, userId int64, keys []model.ItemKey) (map[model.ItemKey]int64, error) {

	for _, key := range keys {
		if _, ok := itemStackByCategory[key.Category]; !ok {
			return nil, protocol.CodeItemInvalidParam
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	itemByItemKey, err := repository.Transact(ctx, s.db, func(store *repository.TxStore) (map[model.ItemKey]*model.Item, error) {
		return loadItems(ctx, store, userId, keys)
	})
	if err != nil {
		return nil, dbFailed(err, "get item counts userId:%d", userId)
	}
	countByItemKey := make(map[model.ItemKey]int64, len(keys))
	for _, key := range keys {
		if item, ok := itemByItemKey[key]; ok {
			countByItemKey[key] = item.Count
		} else {
			countByItemKey[key] = 0
		}
	}
	return countByItemKey, nil
}

func (s *ItemService) Add(ctx context.Context, userId int64, adds []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	return s.updateItem(ctx, userId, nil, adds, reason)
}

func (s *ItemService) Decr(ctx context.Context, userId int64, costs []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	return s.updateItem(ctx, userId, costs, nil, reason)
}

// 先扣后加,任一道具不足则全部不生效
func (s *ItemService) updateItem(ctx context.Context, userId int64, costs, adds []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	res, err := repository.Transact(ctx, s.db, func(store *repository.TxStore) (*model.ItemChangeResult, error) {
		return s.doUpdateItem(ctx, store, userId, costs, adds, reason)
	})
	if err != nil {
		return nil, txFailed(err, "change item userId:%d reason:%d", userId, reason)
	}
	s.NotifyChange(res)
	return res, nil
}

func (s *ItemService) doUpdateItem(ctx context.Context, store *repository.TxStore, userId int64, costs, adds []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {

	//先做一下合并
	costByItemKey, err := mergeItemCounts(costs)
	if err != nil {
		return nil, err
	}
	addByItemKey, err := mergeItemCounts(adds)
	if err != nil {
		return nil, err
	}
	keys := make([]model.ItemKey, 0, len(costByItemKey)+len(addByItemKey))
	for key := range costByItemKey {
		keys = append(keys, key)
	}
	for key := range addByItemKey {
		if _, ok := costByItemKey[key]; !ok {
			keys = append(keys, key)
		}
	}
	res := &model.ItemChangeResult{UserId: userId, Reason: reason}
	if len(keys) == 0 {
		return res, nil
	}

	if _, err = s.lockUser(ctx, store, userId); err != nil {
		return nil, err
	}
	itemByItemKey, err := loadItems(ctx, store, userId, keys)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	changed := make([]*model.Item, 0, len(keys))
	for _, key := range keys {
		item, ok := itemByItemKey[key]
		if !ok {
			item = &model.Item{UserId: userId, Category: key.Category, ItemId: key.ItemId, CreateTime: now}
		}
		before := item.Count
		if before < costByItemKey[key] {
			return nil, protocol.CodeItemNotEnough
		}
		after := before - costByItemKey[key] + addByItemKey[key]
		if after == before {
			continue
		}
		item.Count = after
		item.UpdateTime = now
		changed = append(changed, item)
		res.Changes = append(res.Changes, &model.ItemChange{ItemKey: key, Before: before, After: after})
	}

	if err = store.Item.Upsert(ctx, changed); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *ItemService) NotifyChange(res *model.ItemChangeResult) {
	if res == nil || len(res.Changes) == 0 {
		return
	}
	push := &protocol.ItemChangePush{Reason: res.Reason, Changes: make([]*protocol.ItemChangeInfo, 0, len(res.Changes))}
	for _, c := range res.Changes {
		btlog.Ins().Info("itemChange", "userId", res.UserId, "category", c.Category, "itemId", c.ItemId,
			"before", c.Before, "after", c.After, "reason", res.Reason)
		push.Changes = append(push.Changes, &protocol.ItemChangeInfo{Category: c.Category, ItemId: c.ItemId, Before: c.Before, After: c.After})
	}
	if err := s.pushService.PushUser(res.UserId, protocol.PushItemChange, push); err != nil && !errors.Is(err, errUserOffline) {
		btlog.Ins().Warn("pushItemChangeError", "userId", res.UserId, "err", err)
	}
}

// todo 暂时在用户身上加锁，同时分配 itemSeq 用于道具变更
func (s *ItemService) lockUser(ctx context.Context, store *repository.TxStore, userId int64) (int64, error) {
	itemSeq, exists, err := store.User.GetItemSeqForUpdate(ctx, userId)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, protocol.CodeUserNotFound
	}
	return itemSeq, nil
}

// 校验并合并同一道具的多条数量
func mergeItemCounts(counts []model.ItemChangeReq) (map[model.ItemKey]int64, error) {
	countByItemKey := make(map[model.ItemKey]int64, len(counts))
	for _, c := range counts {
		if c.Count <= 0 {
			return nil, protocol.CodeItemInvalidParam
		}
		if _, ok := itemStackByCategory[c.Category]; !ok {
			return nil, protocol.CodeItemInvalidParam
		}
		countByItemKey[c.ItemKey] += c.Count
	}
	return countByItemKey, nil
}

func loadItems(ctx context.Context, store *repository.TxStore, userId int64, keys []model.ItemKey) (map[model.ItemKey]*model.Item, error) {
	uniqueKeys := make([]model.ItemKey, 0, len(keys))
	seen := make(map[model.ItemKey]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		uniqueKeys = append(uniqueKeys, key)
	}
	items, err := store.Item.ListByKeys(ctx, userId, uniqueKeys)
	if err != nil {
		return nil, err
	}
	itemByItemKey := make(map[model.ItemKey]*model.Item, len(items))
	for _, item := range items {
		if item.InstanceId == 0 {
			itemByItemKey[item.Key()] = item
		}
	}
	return itemByItemKey, nil
}
