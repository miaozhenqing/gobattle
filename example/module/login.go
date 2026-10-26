package module

import (
	"gobattle/btlog"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/example/proto"
	"strconv"
)

type LoginModule struct {
	mgr    *btsession.Manager
	router *btrouter.Router
}

func NewLoginModule(mgr *btsession.Manager, r *btrouter.Router) *LoginModule {
	return &LoginModule{mgr: mgr, router: r}
}

func (m *LoginModule) RegisterRoutes(r *btrouter.Router) {
	btrouter.HandlePublic(r, proto.MsgLogin, m.Login)
}

func (m *LoginModule) Login(ctx *btrouter.RequestContext, req *proto.LoginReq) (*proto.LoginResp, error) {
	// todo 暂时 token 即 uid
	uid, err := strconv.ParseInt(req.Token, 10, 64)
	if err != nil || uid <= 0 {
		return nil, proto.CodeLoginFailed
	}
	old, err := m.mgr.Bind(ctx.Conn(), uid)
	if err != nil {
		return nil, proto.CodeAlreadyLogin
	}
	if old != nil {
		m.kick(old)
	}
	return &proto.LoginResp{UID: uid}, nil
}

func (m *LoginModule) kick(old *btsession.Session) {
	btlog.Ins().Infof("kick, uid:%v connID:%v", old.UID(), old.Conn().GetID())
	_ = m.router.Push(old.Conn(), proto.PushKick, &proto.KickPush{})
	old.Conn().FlushAndStop()
}
