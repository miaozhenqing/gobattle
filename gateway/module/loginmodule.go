package module

import (
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/framework/btsession"
	"gobattle/gateway/service"
	"gobattle/shared/api"
	"gobattle/shared/protocol"
)

type LoginModule struct {
	userApi     api.UserApi
	pushService *service.PushService
	sessionMgr  *btsession.Manager
}

func NewLoginModule(userApi api.UserApi, pushService *service.PushService, sessionMgr *btsession.Manager) *LoginModule {
	return &LoginModule{userApi: userApi, pushService: pushService, sessionMgr: sessionMgr}
}

func (m *LoginModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.HandlePublic(r, protocol.MsgLogin, m.Login)
	btrouter.Handle(r, protocol.MsgHeartBeat, m.HeartBeat)
}

func (m *LoginModule) Login(ctx *btrouter.RequestContext, req *protocol.LoginReq) (*protocol.LoginResp, error) {
	if _, ok := btsession.GetSession(ctx.Conn()); ok {
		return nil, protocol.CodeAlreadyLogin
	}
	rctx := btrpc.WithRequestMeta(ctx, btrpc.RequestMeta{})
	res, err := m.userApi.Login(rctx, &api.LoginReq{Username: req.Username})
	if err != nil {
		return nil, err
	}
	old, err := m.sessionMgr.Bind(ctx.Conn(), res.UserId)
	if err != nil {
		return nil, protocol.CodeAlreadyLogin
	}
	if old != nil {
		m.kick(old)
	}
	btlog.Ins().Info("login", "userId", res.UserId, "username", res.Username, "created", res.IsNew, "connId", ctx.Conn().GetId())
	return &protocol.LoginResp{UserId: res.UserId, Username: res.Username, RoleName: res.RoleName, IsNew: res.IsNew}, nil
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
