package service

import (
	"context"
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

func (s *ShopService) buyItem(ctx context.Context, shopId int) (*ShopBuyItem, error) {
	//商品配置检查
	_, ok := s.tplMgr.GetShopTpl(shopId)
	if !ok {
		return nil, protocol.CodeCommonInvalidParam
	}
	//限购检查
	//扣除消耗
	//发奖
	return nil, nil
}
