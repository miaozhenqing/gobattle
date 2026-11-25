package btsession

import (
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"sync/atomic"
	"time"
)

var sessionKey = btnet.NewAttrKey[*Session]("btsession.session")

// Session 绑定后不可变
type Session struct {
	userId          int64
	conn            *btnet.Connection
	sessionId       string
	version         int64
	onlineCheckedAt atomic.Int64
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
func (s *Session) OnlineCheckedWithin(d time.Duration) bool {
	n := s.onlineCheckedAt.Load()
	if n == 0 || d <= 0 {
		return false
	}
	elapsed := time.Since(time.Unix(0, n))
	return elapsed >= 0 && elapsed < d
}

func (s *Session) SessionId() string {
	return s.sessionId
}
func (s *Session) Version() int64 {
	return s.version
}
func (s *Session) MarkOnlineChecked() {
	s.onlineCheckedAt.Store(time.Now().UnixNano())
}
