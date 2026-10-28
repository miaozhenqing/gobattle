package btsession

import (
	"errors"
	"gobattle/btlog"
	"gobattle/btnet"
	"sync"
)

var ErrAlreadyLogin = errors.New("conn already login")

type Manager struct {
	mu              sync.RWMutex
	sessions        map[int64]*Session
	unbindListeners []func(s *Session)
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[int64]*Session)}
}

func (m *Manager) Bind(conn *btnet.Connection, userId int64) (*Session, error) {
	if _, ok := GetSession(conn); ok {
		return nil, ErrAlreadyLogin
	}
	s := &Session{userId: userId, conn: conn}
	sessionKey.Set(conn, s)

	m.mu.Lock()
	old := m.sessions[userId]
	m.sessions[userId] = s
	m.mu.Unlock()

	btlog.Ins().Info("sessionBind", "userId", userId, "connId", conn.GetID())
	return old, nil
}

func (m *Manager) Get(userId int64) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[userId]
	return s, ok
}

func (m *Manager) unbind(conn *btnet.Connection) {
	s, ok := GetSession(conn)
	if !ok {
		return
	}
	m.mu.Lock()
	// 只删自己，被顶号的旧连接断开时不能删掉新 session
	if m.sessions[s.userId] == s {
		delete(m.sessions, s.userId)
	}
	m.mu.Unlock()
	btlog.Ins().Info("sessionUnbind", "userId", s.userId, "connId", conn.GetID())

	for _, fn := range m.unbindListeners {
		fn(s)
	}
}

func (m *Manager) OnUnbind(fn func(s *Session)) {
	m.unbindListeners = append(m.unbindListeners, fn)
}
