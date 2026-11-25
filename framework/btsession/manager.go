package btsession

import (
	"errors"
	"gobattle/framework/btlog"
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"sync"
)

var ErrAlreadyLogin = errors.New("conn already login")
var ErrLoginSuperseded = errors.New("newer local session exists")
var ErrConnClosed = errors.New("conn closed")

type Manager struct {
	mu              sync.RWMutex
	sessions        map[int64]*Session
	unbindListeners []func(s *Session)
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[int64]*Session)}
}

func (m *Manager) Bind(conn *btnet.Connection, userId int64, id string, version int64) (*Session, *Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := GetSession(conn); ok {
		return nil, nil, ErrAlreadyLogin
	}

	old := m.sessions[userId]
	if old != nil && old.version >= version {
		return nil, nil, ErrLoginSuperseded
	}
	if conn.IsClosed() {
		return nil, nil, ErrConnClosed
	}
	s := &Session{
		userId:    userId,
		conn:      conn,
		sessionId: id,
		version:   version,
	}
	s.MarkOnlineChecked()
	sessionKey.Set(conn, s)
	m.sessions[userId] = s
	btlog.Ins().Info("sessionBind", "userId", userId, "sessionId", id, "version", version, "connId", conn.GetId())
	return s, old, nil
}

func (m *Manager) Get(userId int64) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[userId]
	return s, ok
}

func (m *Manager) IsCurrentSession(s *Session) bool {
	if s == nil || s.conn.IsClosed() {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[s.userId] == s
}

func (m *Manager) IsCurrent(ctx2 *btrouter.RequestContext) bool {
	session, ok := GetSession(ctx2.Conn())
	return ok && m.IsCurrentSession(session)
}

func (m *Manager) Drop(s *Session) {
	if s == nil {
		return
	}
	m.mu.Lock()
	if m.sessions[s.userId] == s {
		delete(m.sessions, s.userId)
	}
	if current, ok := GetSession(s.conn); ok && current == s {
		sessionKey.Remove(s.conn)
	}
	m.mu.Unlock()
}

func (m *Manager) Range(fn func(*Session)) {
	m.mu.RLock()
	list := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, s)
	}
	m.mu.RUnlock()
	for _, s := range list {
		fn(s)
	}
}

func (m *Manager) unbind(conn *btnet.Connection) {
	s, ok := GetSession(conn)
	if !ok {
		return
	}
	m.mu.Lock()
	if m.sessions[s.userId] == s {
		delete(m.sessions, s.userId)
	}
	m.mu.Unlock()
	btlog.Ins().Info("sessionUnbind", "userId", s.userId, "sessionId", s.sessionId, "connId", conn.GetId())
	for _, fn := range m.unbindListeners {
		fn(s)
	}
}

func (m *Manager) OnUnbind(fn func(*Session)) { m.unbindListeners = append(m.unbindListeners, fn) }
