package service

import (
	"context"
	"gobattle/btsession"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"gobattle/game/tpl"
	"time"
)

type ShopService struct {
	db           *repository.DB
	tplMgr       *tpl.TplManager
	queryTimeout time.Duration
}

type ShopBuyItem struct {
}

func (s *ShopService) buyItem(ctx context.Context, session *btsession.Session, shopId int) (*ShopBuyItem, error) {
	//商品配置检查
	shopTpl, ok := s.tplMgr.GetShopTpl(shopId)
	if !ok {
		return nil, protocol.CodeCommonInvalidParam
	}
	userId := session.UserId()
	//限购检查
	shop, err := s.getUserShopOrInsert(ctx, userId)
	if err != nil {
		return nil, err
	}
	if shop.BuyCount >= shopTpl.BuyLimit {
		return nil, protocol.CodeCommonLimitExceed
	}

	//扣除消耗 todo
	//发奖 todo
	return nil, nil
}

// 获取用户商店，无则创建；//todo 统一方法，会嵌入刷新逻辑
func (s *ShopService) getUserShopOrInsert(ctx context.Context, userId int64) (*model.UserShop, error) {
	//todo 从数据库中查询用户商店,无则创建
	//todo 通过for update 加锁
	return nil, nil
}
