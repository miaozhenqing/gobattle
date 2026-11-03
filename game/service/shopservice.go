package service

import (
	"context"
	"errors"
	"gobattle/btsession"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"gobattle/game/tpl"
	"time"

	"github.com/bwmarrin/snowflake"
)

type ShopService struct {
	db           *repository.DB
	tplMgr       *tpl.TplManager
	itemService  *ItemService
	idNode       *snowflake.Node
	queryTimeout time.Duration
}

func NewShopService(db *repository.DB, tplMgr *tpl.TplManager, itemService *ItemService, idNode *snowflake.Node, queryTimeout time.Duration) *ShopService {
	return &ShopService{db: db, tplMgr: tplMgr, itemService: itemService, idNode: idNode, queryTimeout: queryTimeout}
}

type ShopBuyItem struct {
}

func (s *ShopService) buyItem(ctx context.Context, session *btsession.Session, shopId int) (*ShopBuyItem, error) {
	//商品配置检查
	shopTpl, ok := s.tplMgr.GetShopTpl(shopId)
	if !ok {
		return nil, protocol.CodeCommonInvalidParam
	}
	costs, adds, err := parseCostAndReward(shopTpl)
	if err != nil {
		return nil, err
	}
	userId := session.UserId()

	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	_, err = repository.Transact(ctx, s.db, func(ctx context.Context, store *repository.TxStore) (struct{}, error) {
		now := time.Now().UnixMilli()
		shop, err := s.getUserShopOrInsert(ctx, store, userId, shopId, now)
		if err != nil {
			return struct{}{}, err
		}
		// 限购检查
		if shop.BuyCount >= shopTpl.BuyLimit {
			return struct{}{}, protocol.CodeCommonLimitExceed
		}
		// 扣消耗
		if _, err := s.itemService.Decr(ctx, userId, costs, model.ItemReasonShopBuy); err != nil {
			return struct{}{}, err
		}
		//发奖
		if _, err := s.itemService.Add(ctx, userId, adds, model.ItemReasonShopBuy); err != nil {
			return struct{}{}, err
		}
		shop.BuyCount++
		shop.LastBuyTime = now
		shop.UpdateTime = now
		return struct{}{}, store.Shop.Update(ctx, shop)
	})
	if err != nil {
		return nil, txFailed(err, "buy shop userId:%d shopId:%d", userId, shopId)
	}
	return &ShopBuyItem{}, nil
}

// 获取用户商品记录并加锁，无则创建；todo 统一方法，会嵌入刷新逻辑
func (s *ShopService) getUserShopOrInsert(ctx context.Context, store *repository.TxStore, userId int64, shopId int, now int64) (*model.UserShop, error) {
	shop, err := store.Shop.GetByUserIdAndShopIdForUpdate(ctx, userId, shopId)
	if err != nil || shop != nil {
		return shop, err
	}
	shop = &model.UserShop{Id: s.idNode.Generate().Int64(), UserId: userId, ShopId: shopId, CreateTime: now, UpdateTime: now}
	return shop, store.Shop.Insert(ctx, shop)
}

func parseCostAndReward(shopTpl *tpl.ShopTpl) (costs, adds []model.ItemChangeReq, err error) {
	return nil, nil, errors.New("parseCostAndReward not implemented")
}
