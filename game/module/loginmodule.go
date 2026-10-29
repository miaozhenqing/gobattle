package module

import (
	"context"
	"gobattle/btlog"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/proto"
	"unicode/utf8"
)

type LoginModule struct {
	*Deps
}

func NewLoginModule(d *Deps) *LoginModule {
	return &LoginModule{Deps: d}
}

func (m *LoginModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.HandlePublic(r, proto.MsgLogin, m.Login)
	btrouter.Handle(r, proto.MsgHeartBeat, m.HeartBeat)
}

func (m *LoginModule) Login(ctx *btrouter.RequestContext, req *proto.LoginReq) (*proto.LoginResp, error) {
	if !validUsername(req.Username) {
		return nil, proto.CodeInvalidUsername
	}
	if _, ok := btsession.GetSession(ctx.Conn()); ok {
		return nil, proto.CodeAlreadyLogin
	}

	dbCtx, cancel := context.WithTimeout(ctx, m.QueryTimeout)
	defer cancel()
	res, err := m.UserService.Login(dbCtx, req.Username)
	if err != nil {
		return nil, err
	}
	user := res.User

	old, err := m.SessionMgr.Bind(ctx.Conn(), user.UserId)
	if err != nil {
		return nil, proto.CodeAlreadyLogin
	}
	if old != nil {
		m.kick(old)
	}
	btlog.Ins().Info("login", "userId", user.UserId, "username", user.Username, "created", res.Created, "connId", ctx.Conn().GetID())
	return &proto.LoginResp{UserId: user.UserId, Username: user.Username, RoleName: user.RoleName, IsNew: res.Created}, nil
}

func (m *LoginModule) kick(old *btsession.Session) {
	btlog.Ins().Info("kick", "userId", old.UserId(), "connId", old.Conn().GetID())
	_ = m.Router.Push(old.Conn(), proto.PushKick, &proto.KickPush{})
	old.Conn().FlushAndStop()
}

func (m *LoginModule) HeartBeat(requestContext *btrouter.RequestContext, req *proto.HeartBeatReq) (*proto.HeartBeatResp, error) {
	if !btsession.IsLoggedIn(requestContext) {
		return nil, proto.CodeNotLogin
	}
	return &proto.HeartBeatResp{UserId: req.UserId}, nil
}

func validUsername(s string) bool {
	if s == "" || !utf8.ValidString(s) {
		return false
	}
	return utf8.RuneCountInString(s) <= 32
}
