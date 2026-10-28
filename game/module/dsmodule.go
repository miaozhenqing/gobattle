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
	dsById              map[int64]*model.DsModel
	mu                  sync.RWMutex
	unregisterListeners []func(ds *model.DsModel)
}

func NewDsModule(d *Deps) *DsModule {
	return &DsModule{Deps: d, dsById: make(map[int64]*model.DsModel)}
}

func (m *DsModule) RegisterRoutes(r *btrouter.Router) {
	btrouter.HandlePublic(r, proto.MsgDsRegister, m.Register)
	btrouter.HandlePublic(r, proto.MsgDsHeartBeat, m.HeartBeat)
}

func (m *DsModule) OnUnregister(fn func(ds *model.DsModel)) {
	m.unregisterListeners = append(m.unregisterListeners, fn)
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
	old := m.dsById[ds.ID]
	m.dsById[ds.ID] = ds
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

func (m *DsModule) unregister(ds *model.DsModel) {
	m.mu.Lock()
	if m.dsById[ds.ID] == ds {
		delete(m.dsById, ds.ID)
	}
	m.mu.Unlock()
	btlog.Ins().Info("dsUnregister", "dsId", ds.ID, "connId", ds.Conn.GetID())

	for _, fn := range m.unregisterListeners {
		fn(ds)
	}
}

func GetDs(conn *btnet.Connection) (*model.DsModel, bool) {
	return dsKey.Get(conn)
}

// 先随机选一个可用的
func (m *DsModule) PickDs() *model.DsModel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, dsModel := range m.dsById {
		if tryOccupy(dsModel) {
			return dsModel
		}
	}
	return nil
}

func (m *DsModule) ReleaseDs(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateBusy, model.DsStateIdle)
}

func tryOccupy(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateIdle, model.DsStateBusy)
}
