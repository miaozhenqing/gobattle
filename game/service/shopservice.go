package service

import (
	"context"
	"errors"
	"gobattle/btsession"
	"gobattle/game/domain"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"gobattle/game/tpl"
	"gobattle/game/util/timeutil"
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

func (s *ShopService) Buy(ctx context.Context, session *btsession.Session, shopId int) (*domain.ShopPurchase, error) {
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
	res, err := repository.Transact(ctx, s.db, func(ctx context.Context, store *repository.TxStore) (*domain.ShopPurchase, error) {
		now := timeutil.Now().UnixMilli()
		shop, err := s.getUserShopOrInsert(ctx, store, userId, shopTpl, now)
		if err != nil {
			return nil, err
		}
		// 限购检查
		if shopTpl.BuyLimit > 0 && shop.BuyCount >= shopTpl.BuyLimit {
			return nil, protocol.CodeCommonLimitExceed
		}
		// 扣消耗
		_, err = s.itemService.Decr(ctx, userId, costs, domain.ItemReasonShopBuy)
		if err != nil {
			return nil, err
		}
		//发奖
		_, err = s.itemService.Add(ctx, userId, adds, domain.ItemReasonShopBuy)
		if err != nil {
			return nil, err
		}
		shop.BuyCount++
		shop.LastBuyTime = now
		shop.UpdateTime = now
		if err := store.ShopRepository.Update(ctx, shop); err != nil {
			return nil, err
		}
		return &domain.ShopPurchase{Shop: shop}, nil
	})
	if err != nil {
		return nil, txFailed(err, "buy shop userId:%d shopId:%d", userId, shopId)
	}
	return res, nil
}

// 统一方法:获取用户商品记录并加锁，无则创建；
func (s *ShopService) getUserShopOrInsert(ctx context.Context, store *repository.TxStore, userId int64, shopTpl *tpl.ShopTpl, now int64) (*model.UserShop, error) {
	shopId := shopTpl.Id
	shop, err := store.ShopRepository.GetByUserIdAndShopIdForUpdate(ctx, userId, shopId)
	if err != nil {
		return nil, err
	}
	if shop != nil {
		//刷新
		refresh(shop, shopTpl)
		return shop, nil
	}
	shop = &model.UserShop{Id: s.idNode.Generate().Int64(), UserId: userId, ShopId: shopId, CreateTime: now, UpdateTime: now}
	return shop, store.ShopRepository.Insert(ctx, shop)
}

func refresh(shop *model.UserShop, shopTpl *tpl.ShopTpl) {
	if shop == nil {
		return
	}
	now := timeutil.Now()
	switch shopTpl.Refresh {
	case tpl.RefreshDaily:
		if !timeutil.IsSameDay(shop.LastBuyTime, now.UnixMilli()) {
			shop.BuyCount = 0
		}
	case tpl.RefreshWeekly:
		if !timeutil.IsSameWeek(shop.LastBuyTime, now.UnixMilli()) {
			shop.BuyCount = 0
		}
	case tpl.RefreshMonthly:
		if !timeutil.IsSameMonth(shop.LastBuyTime, now.UnixMilli()) {
			shop.BuyCount = 0
		}
	default:
		break
	}
}

func parseCostAndReward(shopTpl *tpl.ShopTpl) (costs, adds []domain.ItemAmount, err error) {
	//todo
	return nil, nil, errors.New("parseCostAndReward not implemented")
}
