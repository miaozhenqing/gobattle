package module

import (
	"gobattle/common/api"
	"gobattle/common/protocol"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/framework/btsession"
	"gobattle/gateway/service"
)

type LoginModule struct {
	userApi       api.UserApi
	sessionMgr    *btsession.Manager
	onlineService *service.OnlineService
	kick          *service.KickService
}

func NewLoginModule(userApi api.UserApi, sessionMgr *btsession.Manager,
	onlineService *service.OnlineService, kick *service.KickService) *LoginModule {
	return &LoginModule{userApi: userApi, sessionMgr: sessionMgr, onlineService: onlineService, kick: kick}
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
	res, err := m.userApi.Login(rctx, req)
	if err != nil {
		return nil, err
	}
	online, err := m.onlineService.SetOnline(ctx, res.UserId)
	if err != nil {
		return nil, protocol.CodeLoginUnavailable
	}
	s, old, err := m.sessionMgr.Bind(ctx.Conn(), res.UserId, online.SessionId, online.Version)
	if err != nil {
		m.onlineService.RemoveOnline(online)
		return nil, protocol.CodeLoginSuperseded
	}
	if old != nil {
		m.kick.KickSession(old)
	}
	current, exists, err := m.onlineService.GetOnline(ctx, res.UserId)
	if err != nil || !exists || current.GatewayId != online.GatewayId ||
		current.SessionId != online.SessionId || current.Version != online.Version || ctx.Conn().IsClosed() {
		m.sessionMgr.Drop(s)
		m.onlineService.RemoveOnline(online)
		if err != nil {
			return nil, protocol.CodeLoginUnavailable
		}
		return nil, protocol.CodeLoginSuperseded
	}
	s.MarkOnlineChecked()
	btlog.Ins().Info("login", "userId", res.UserId, "sessionId", s.SessionId(),
		"version", s.Version(), "username", res.Username, "created", res.IsNew, "connId", ctx.Conn().GetId())
	return res, nil
}

func (m *LoginModule) HeartBeat(requestContext *btrouter.RequestContext, req *protocol.HeartBeatReq) (*protocol.HeartBeatResp, error) {
	if !m.onlineService.IsActiveSession(requestContext) {
		return nil, protocol.CodeNotLogin
	}
	return &protocol.HeartBeatResp{UserId: req.UserId}, nil
}
