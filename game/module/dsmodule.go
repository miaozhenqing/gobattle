package module

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/game/model"
	"gobattle/game/proto"
	"net"
	"sync"
	"time"
)

//注册ds服务器，暂时和用户一个端口

var dsKey = btnet.NewAttrKey[*model.DsModel]("module.ds")

type DsModule struct {
	*Deps
	dsServers map[int64]*model.DsModel
	mu        sync.RWMutex
}

func NewDsModule(d *Deps) *DsModule {
	return &DsModule{Deps: d, dsServers: make(map[int64]*model.DsModel)}
}

func (m *DsModule) RegisterRoutes(r *btrouter.Router) {
	btrouter.HandlePublic(r, proto.MsgDsRegister, m.Register)
	btrouter.HandlePublic(r, proto.MsgDsHeartBeat, m.HeartBeat)
}

func (m *DsModule) Register(ctx *btrouter.RequestContext, req *proto.DsRegisterReq) (*proto.DsRegisterResp, error) {
	if net.ParseIP(req.IP) == nil || req.Port <= 0 || req.DsID <= 0 {
		return nil, proto.CodeDsInvalidParam
	}
	conn := ctx.Conn()
	if _, ok := dsKey.Get(conn); ok {
		return nil, proto.CodeDsAlreadyRegistered
	}

	ds := &model.DsModel{
		ID:           req.DsID,
		IP:           req.IP,
		Port:         req.Port,
		RegisterTime: time.Now().UnixMilli(),
		Conn:         conn,
	}
	dsKey.Set(conn, ds)

	m.mu.Lock()
	old := m.dsServers[ds.ID]
	m.dsServers[ds.ID] = ds
	m.mu.Unlock()
	if old != nil {
		btlog.Ins().Infof("ds replaced, dsID:%v oldConnID:%v newConnID:%v", ds.ID, old.Conn.GetID(), conn.GetID())
		old.Conn.FlushAndStop()
	}

	go func() {
		<-conn.Context().Done()
		m.unregister(ds)
	}()

	btlog.Ins().Infof("ds register, dsID:%v addr:%v:%v connID:%v", ds.ID, ds.IP, ds.Port, conn.GetID())
	return &proto.DsRegisterResp{DsID: ds.ID}, nil
}

func (m *DsModule) HeartBeat(ctx *btrouter.RequestContext, _ *proto.DsHeartBeatReq) (*proto.DsHeartBeatResp, error) {
	ds, ok := dsKey.Get(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	return &proto.DsHeartBeatResp{DsID: ds.ID}, nil
}

func (m *DsModule) unregister(ds *model.DsModel) {
	m.mu.Lock()
	if m.dsServers[ds.ID] == ds {
		delete(m.dsServers, ds.ID)
	}
	m.mu.Unlock()
	btlog.Ins().Infof("ds unregister, dsID:%v connID:%v", ds.ID, ds.Conn.GetID())
}
