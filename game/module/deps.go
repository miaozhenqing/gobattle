package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/service"
	"time"
)

type Deps struct {
	Router       *btrouter.Router
	SessionMgr   *btsession.Manager
	QueryTimeout time.Duration

	// 所有 service，新增 service 加在 service.Deps 里
	*service.Deps
}
