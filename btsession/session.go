package btsession

import (
	"gobattle/btnet"
	"gobattle/btrouter"
)

var sessionKey = btnet.NewAttrKey[*Session]("btsession.session")

// Session 绑定后不可变
type Session struct {
	userId int64
	conn   *btnet.Connection
}

func (s *Session) UserId() int64 {
	return s.userId
}

func (s *Session) Conn() *btnet.Connection {
	return s.conn
}

func GetSession(conn *btnet.Connection) (*Session, bool) {
	return sessionKey.Get(conn)
}

func IsLoggedIn(ctx *btrouter.RequestContext) bool {
	_, ok := GetSession(ctx.Conn())
	return ok
}
