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
	btrouter.HandlePublic(r, proto.MsgDsRelease, m.Release)
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
		btlog.Ins().Info("dsReplaced", "dsId", ds.ID, "oldConnId", old.Conn.GetID(), "newConnId", conn.GetID())
		old.Conn.FlushAndStop()
	}

	go func() {
		<-conn.Context().Done()
		m.unregister(ds)
	}()

	btlog.Ins().Info("dsRegister", "dsId", ds.ID, "ip", ds.IP, "port", ds.Port, "connId", conn.GetID())
	return &proto.DsRegisterResp{DsID: ds.ID}, nil
}

func (m *DsModule) HeartBeat(ctx *btrouter.RequestContext, _ *proto.DsHeartBeatReq) (*proto.DsHeartBeatResp, error) {
	ds, ok := dsKey.Get(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	return &proto.DsHeartBeatResp{DsID: ds.ID}, nil
}

func (m *DsModule) Release(ctx *btrouter.RequestContext, req *proto.DsReleaseReq) (*proto.DsReleaseResp, error) {
	ds, ok := dsKey.Get(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	if !tryRelease(ds) {
		btlog.Ins().Warn("dsReleaseAlreadyIdle", "dsId", ds.ID, "battleId", req.BattleId, "connId", ds.Conn.GetID())
	} else {
		btlog.Ins().Info("dsRelease", "dsId", ds.ID, "battleId", req.BattleId, "connId", ds.Conn.GetID())
	}
	return &proto.DsReleaseResp{DsID: ds.ID}, nil
}

func (m *DsModule) unregister(ds *model.DsModel) {
	m.mu.Lock()
	if m.dsServers[ds.ID] == ds {
		delete(m.dsServers, ds.ID)
	}
	m.mu.Unlock()
	btlog.Ins().Info("dsUnregister", "dsId", ds.ID, "connId", ds.Conn.GetID())
}

// 先随机选一个可用的
func (m *DsModule) PickDs() *model.DsModel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, dsModel := range m.dsServers {
		if tryOccupy(dsModel) {
			return dsModel
		}
	}
	return nil
}

func tryOccupy(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateIdle, model.DsStateBusy)
}

func tryRelease(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateBusy, model.DsStateIdle)
}
