package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/domain"
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
	userItems, err := m.itemService.List(ctx, s.UserId())
	if err != nil {
		return nil, err
	}
	items := make([]*protocol.ItemInfo, 0, len(userItems))
	for _, it := range userItems {
		items = append(items, &protocol.ItemInfo{Id: it.Id, Category: int(it.Category), ItemId: it.ItemId, Count: it.Count})
	}
	return &protocol.ItemListResp{Items: items}, nil
}

func toItemChangeInfos(changes []*domain.ItemChange) []*protocol.ItemChangeInfo {
	infos := make([]*protocol.ItemChangeInfo, 0, len(changes))
	for _, c := range changes {
		infos = append(infos, &protocol.ItemChangeInfo{
			ItemInfo: protocol.ItemInfo{Id: c.Id, Category: int(c.Category), ItemId: c.ItemId, Count: c.After}, Before: c.Before})
	}
	return infos
}
