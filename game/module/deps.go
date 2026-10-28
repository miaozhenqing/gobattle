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

	DsModule *DsModule

	*service.Deps
}
