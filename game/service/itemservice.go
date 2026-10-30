package service

import (
	"context"
	"errors"
	"fmt"
	"gobattle/btlog"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"gobattle/game/service/item"
	"time"

	"github.com/bwmarrin/snowflake"
)

// 道具业务
type ItemService struct {
	db           *repository.DB
	pushService  *PushService
	idNode       *snowflake.Node
	queryTimeout time.Duration
}

func NewItemService(db *repository.DB, pushService *PushService, idNode *snowflake.Node, queryTimeout time.Duration) *ItemService {

	for _, def := range model.ItemCategoryDefs() {
		_, hasStack := item.GetStackHandler(def.Category)
		_, hasUnstack := item.GetUnstackHandler(def.Category)
		if def.Stackable != hasStack || def.Stackable == hasUnstack {
			panic(fmt.Sprintf("item category %d handler not match stackable:%v", def.Category, def.Stackable))
		}
	}
	return &ItemService{db: db, pushService: pushService, idNode: idNode, queryTimeout: queryTimeout}
}

// 仓库道具列表
func (s *ItemService) List(ctx context.Context, userId int64) ([]*protocol.ItemInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	items, err := repository.Transact(ctx, s.db, func(store *repository.TxStore) ([]*protocol.ItemInfo, error) {
		var items []*protocol.ItemInfo
		//todo 优化查询
		for _, def := range model.ItemCategoryDefs() {
			bases, err := item.GetHandler(def.Category).List(ctx, store, userId)
			if err != nil {
				return nil, err
			}
			for _, b := range bases {
				items = append(items, newItemInfo(def.Category, b.Id, b.ItemId, b.Count))
			}
		}
		return items, nil
	})
	if err != nil {
		return nil, dbFailed(err, "list item userId:%d", userId)
	}
	return items, nil
}

func newItemInfo(category model.ItemCategory, id int64, itemId int, count int64) *protocol.ItemInfo {
	return &protocol.ItemInfo{Id: id, Category: int(category), ItemId: itemId, Count: count}
}

func (s *ItemService) Add(ctx context.Context, userId int64, adds []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	return s.updateItem(ctx, userId, nil, adds, reason)
}

func (s *ItemService) Decr(ctx context.Context, userId int64, costs []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	return s.updateItem(ctx, userId, costs, nil, reason)
}

// 先扣后加,任一道具不足则全部不生效
func (s *ItemService) updateItem(ctx context.Context, userId int64, costs []model.ItemChangeReq, adds []model.ItemChangeReq, reason int) (*model.ItemChangeResult, error) {
	ops, err := mergeItemOps(costs, adds)
	if err != nil {
		return nil, err
	}
	res := &model.ItemChangeResult{UserId: userId, Reason: reason}
	if ops.empty() {
		return res, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	res, err = repository.Transact(ctx, s.db, func(store *repository.TxStore) (*model.ItemChangeResult, error) {
		return s.doUpdateItem(ctx, store, res, ops)
	})
	if err != nil {
		return nil, txFailed(err, "change item userId:%d reason:%d", userId, reason)
	}
	s.NotifyChange(res)
	return res, nil
}

func (s *ItemService) doUpdateItem(ctx context.Context, store *repository.TxStore, res *model.ItemChangeResult, ops *itemOps) (*model.ItemChangeResult, error) {
	userId := res.UserId
	if err := s.lockUser(ctx, store, userId); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()

	// 不可堆叠扣减
	for category, itemIdById := range ops.unstackCosts {
		handler, _ := item.GetUnstackHandler(category)
		if err := handler.Delete(ctx, store, userId, itemIdById); err != nil {
			return nil, err
		}
		for id, itemId := range itemIdById {
			res.Changes = append(res.Changes, &model.ItemChange{ItemKey: model.ItemKey{Category: category, ItemId: itemId}, Id: id, Before: 1, After: 0})
		}
	}

	// 可堆叠增减
	for category, deltaByItemId := range ops.stackDeltas {
		handler, _ := item.GetStackHandler(category)
		changes, err := handler.UpdateCount(ctx, store, userId, deltaByItemId, s.newId, now)
		if err != nil {
			return nil, err
		}
		res.Changes = append(res.Changes, changes...)
	}

	// 不可堆叠增加
	for category, countByItemId := range ops.unstackAdds {
		handler, _ := item.GetUnstackHandler(category)
		for itemId, count := range countByItemId {
			ids := make([]int64, count)
			for i := range ids {
				ids[i] = s.newId()
			}
			if err := handler.Add(ctx, store, userId, itemId, ids, now); err != nil {
				return nil, err
			}
			for _, id := range ids {
				res.Changes = append(res.Changes, &model.ItemChange{ItemKey: model.ItemKey{Category: category, ItemId: itemId}, Id: id, Before: 0, After: 1})
			}
		}
	}
	return res, nil
}

func (s *ItemService) NotifyChange(res *model.ItemChangeResult) {
	if res == nil || len(res.Changes) == 0 {
		return
	}
	push := &protocol.ItemChangePush{Reason: res.Reason, Changes: make([]*protocol.ItemChangeInfo, 0, len(res.Changes))}
	for _, c := range res.Changes {
		btlog.Ins().Info("itemChange", "userId", res.UserId, "category", c.Category, "itemId", c.ItemId, "id", c.Id,
			"before", c.Before, "after", c.After, "reason", res.Reason)
		push.Changes = append(push.Changes, &protocol.ItemChangeInfo{
			ItemInfo: *newItemInfo(c.Category, c.Id, c.ItemId, c.After), Before: c.Before})
	}
	if err := s.pushService.PushUser(res.UserId, protocol.PushItemChange, push); err != nil && !errors.Is(err, errUserOffline) {
		btlog.Ins().Warn("pushItemChangeError", "userId", res.UserId, "err", err)
	}
}

func (s *ItemService) newId() int64 {
	return s.idNode.Generate().Int64()
}

// todo 暂时在用户身上加锁，串行化该玩家的道具变更
func (s *ItemService) lockUser(ctx context.Context, store *repository.TxStore, userId int64) error {
	exists, err := store.User.LockForUpdate(ctx, userId)
	if err != nil {
		return err
	}
	if !exists {
		return protocol.CodeUserNotFound
	}
	return nil
}

// 校验并按类型分组后的增减请求
type itemOps struct {
	stackDeltas  map[model.ItemCategory]map[int]*item.StackDelta // 可堆叠：itemId → 增减量
	unstackCosts map[model.ItemCategory]map[int64]int            // 不可堆叠扣减：id → itemId
	unstackAdds  map[model.ItemCategory]map[int]int64            // 不可堆叠增加：itemId → 件数
}

func (o *itemOps) empty() bool {
	return len(o.stackDeltas) == 0 && len(o.unstackCosts) == 0 && len(o.unstackAdds) == 0
}

func (o *itemOps) stackDelta(key model.ItemKey) *item.StackDelta {
	deltaByItemId, ok := o.stackDeltas[key.Category]
	if !ok {
		deltaByItemId = make(map[int]*item.StackDelta)
		o.stackDeltas[key.Category] = deltaByItemId
	}
	delta, ok := deltaByItemId[key.ItemId]
	if !ok {
		delta = &item.StackDelta{}
		deltaByItemId[key.ItemId] = delta
	}
	return delta
}

func mergeItemOps(costs, adds []model.ItemChangeReq) (*itemOps, error) {
	ops := &itemOps{
		stackDeltas:  make(map[model.ItemCategory]map[int]*item.StackDelta),
		unstackCosts: make(map[model.ItemCategory]map[int64]int),
		unstackAdds:  make(map[model.ItemCategory]map[int]int64),
	}
	seenIds := make(map[int64]struct{}, len(costs))
	for _, c := range costs {
		def, ok := model.GetItemCategoryDef(c.Category)
		if !ok || c.Count <= 0 {
			return nil, protocol.CodeItemInvalidParam
		}
		if def.Stackable {
			if c.Id != 0 {
				return nil, protocol.CodeItemInvalidParam
			}
			ops.stackDelta(c.ItemKey).Cost += c.Count
			continue
		}
		if c.Id <= 0 || c.Count != 1 {
			return nil, protocol.CodeItemInvalidParam
		}
		if _, dup := seenIds[c.Id]; dup {
			return nil, protocol.CodeItemInvalidParam
		}
		seenIds[c.Id] = struct{}{}
		itemIdById, ok := ops.unstackCosts[c.Category]
		if !ok {
			itemIdById = make(map[int64]int)
			ops.unstackCosts[c.Category] = itemIdById
		}
		itemIdById[c.Id] = c.ItemId
	}
	for _, c := range adds {
		def, ok := model.GetItemCategoryDef(c.Category)
		if !ok || c.Count <= 0 || c.Id != 0 {
			return nil, protocol.CodeItemInvalidParam
		}
		if def.Stackable {
			ops.stackDelta(c.ItemKey).Add += c.Count
			continue
		}
		countByItemId, ok := ops.unstackAdds[c.Category]
		if !ok {
			countByItemId = make(map[int]int64)
			ops.unstackAdds[c.Category] = countByItemId
		}
		countByItemId[c.ItemId] += c.Count
	}
	return ops, nil
}
