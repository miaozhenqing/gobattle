package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/service"
	"time"
)

type Deps struct {
	// 客户端连接的路由
	GameRouterHandler *btrouter.RouterHandler
	// ds连接的路由
	DsRouterHandler *btrouter.RouterHandler
	SessionMgr      *btsession.Manager
	QueryTimeout    time.Duration

	// service
	UserService *service.UserService
	ItemService *service.ItemService
}
