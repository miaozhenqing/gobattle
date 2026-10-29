package module

import (
	"gobattle/btlog"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/protocol"
	"gobattle/game/service"
	"unicode/utf8"
)

type LoginModule struct {
	userService *service.UserService
	pushService *service.PushService
	sessionMgr  *btsession.Manager
}

func NewLoginModule(userService *service.UserService, pushService *service.PushService, sessionMgr *btsession.Manager) *LoginModule {
	return &LoginModule{userService: userService, pushService: pushService, sessionMgr: sessionMgr}
}

func (m *LoginModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.HandlePublic(r, protocol.MsgLogin, m.Login)
	btrouter.Handle(r, protocol.MsgHeartBeat, m.HeartBeat)
}

func (m *LoginModule) Login(ctx *btrouter.RequestContext, req *protocol.LoginReq) (*protocol.LoginResp, error) {
	if !validUsername(req.Username) {
		return nil, protocol.CodeInvalidUsername
	}
	if _, ok := btsession.GetSession(ctx.Conn()); ok {
		return nil, protocol.CodeAlreadyLogin
	}

	res, err := m.userService.Login(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	user := res.User

	old, err := m.sessionMgr.Bind(ctx.Conn(), user.UserId)
	if err != nil {
		return nil, protocol.CodeAlreadyLogin
	}
	if old != nil {
		m.kick(old)
	}
	btlog.Ins().Info("login", "userId", user.UserId, "username", user.Username, "created", res.Created, "connId", ctx.Conn().GetId())
	return &protocol.LoginResp{UserId: user.UserId, Username: user.Username, RoleName: user.RoleName, IsNew: res.Created}, nil
}

func (m *LoginModule) kick(old *btsession.Session) {
	btlog.Ins().Info("kick", "userId", old.UserId(), "connId", old.Conn().GetId())
	_ = m.pushService.PushSession(old, protocol.PushKick, &protocol.KickPush{})
	old.Conn().FlushAndStop()
}

func (m *LoginModule) HeartBeat(requestContext *btrouter.RequestContext, req *protocol.HeartBeatReq) (*protocol.HeartBeatResp, error) {
	if !btsession.IsLoggedIn(requestContext) {
		return nil, protocol.CodeNotLogin
	}
	return &protocol.HeartBeatResp{UserId: req.UserId}, nil
}

func validUsername(s string) bool {
	if s == "" || !utf8.ValidString(s) {
		return false
	}
	return utf8.RuneCountInString(s) <= 32
}
