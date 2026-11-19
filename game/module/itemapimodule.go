package module

import (
	"context"
	"gobattle/framework/btrpc"
	"gobattle/game/service"
	"gobattle/shared/api"
	"gobattle/shared/protocol"
)

var _ api.ItemApi = (*ItemApiModule)(nil)

type ItemApiModule struct {
	itemService *service.ItemService
}

func NewItemApiModule(itemService *service.ItemService) *ItemApiModule {
	return &ItemApiModule{itemService: itemService}
}

func (m *ItemApiModule) List(ctx context.Context, _ *protocol.ItemListReq) (*protocol.ItemListResp, error) {
	userItems, err := m.itemService.List(ctx, btrpc.UserId(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]*protocol.ItemInfo, 0, len(userItems))
	for _, it := range userItems {
		items = append(items, &protocol.ItemInfo{Id: it.Id, Category: int(it.Category), ItemId: it.ItemId, Count: it.Count})
	}
	return &protocol.ItemListResp{Items: items}, nil
}
