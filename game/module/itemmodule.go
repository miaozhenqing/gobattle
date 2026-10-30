package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/protocol"
	"gobattle/game/service"
)

type ItemModule struct {
	itemService *service.ItemService
}

func NewItemModule(itemService *service.ItemService) *ItemModule {
	return &ItemModule{itemService: itemService}
}

func (m *ItemModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgItemList, m.ListItem)
}

func (m *ItemModule) ListItem(ctx *btrouter.RequestContext, _ *protocol.ItemListReq) (*protocol.ItemListResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, protocol.CodeNotLogin
	}
	items, err := m.itemService.List(ctx, s.UserId())
	if err != nil {
		return nil, err
	}
	resp := &protocol.ItemListResp{Items: make([]*protocol.ItemInfo, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, &protocol.ItemInfo{Category: item.Category, ItemId: item.ItemId, Count: item.Count})
	}
	return resp, nil
}
