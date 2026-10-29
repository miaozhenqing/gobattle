package service

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/game/model"
	"gobattle/game/proto"
	"sync"
	"time"
)

var dsKey = btnet.NewAttrKey[*model.DsModel]("service.ds")

func GetDs(conn *btnet.Connection) (*model.DsModel, bool) {
	return dsKey.Get(conn)
}

type DsService struct {
	mu                  sync.RWMutex
	dsById              map[int64]*model.DsModel
	unregisterListeners []func(ds *model.DsModel)
}

func NewDsService() *DsService {
	return &DsService{dsById: make(map[int64]*model.DsModel)}
}

func (s *DsService) OnUnregister(fn func(ds *model.DsModel)) {
	s.unregisterListeners = append(s.unregisterListeners, fn)
}

func (s *DsService) Register(conn *btnet.Connection, dsId int64, ip string, port int) (*model.DsModel, error) {
	if _, ok := dsKey.Get(conn); ok {
		return nil, proto.CodeDsAlreadyRegistered
	}

	ds := &model.DsModel{
		ID:           dsId,
		IP:           ip,
		Port:         port,
		RegisterTime: time.Now().UnixMilli(),
		Conn:         conn,
	}
	dsKey.Set(conn, ds)

	s.mu.Lock()
	old := s.dsById[ds.ID]
	s.dsById[ds.ID] = ds
	s.mu.Unlock()
	if old != nil {
		btlog.Ins().Info("dsReplaced", "dsId", ds.ID, "oldConnId", old.Conn.GetID(), "newConnId", conn.GetID())
		old.Conn.FlushAndStop()
	}

	btlog.Ins().Info("dsRegister", "dsId", ds.ID, "ip", ds.IP, "port", ds.Port, "connId", conn.GetID())
	return ds, nil
}

func (s *DsService) Unregister(conn *btnet.Connection) {
	ds, ok := dsKey.Get(conn)
	if !ok {
		return
	}
	s.mu.Lock()
	// 只删自己，被顶替的旧连接断开时不能删掉新
	if s.dsById[ds.ID] == ds {
		delete(s.dsById, ds.ID)
	}
	s.mu.Unlock()
	btlog.Ins().Info("dsUnregister", "dsId", ds.ID, "connId", conn.GetID())

	for _, fn := range s.unregisterListeners {
		fn(ds)
	}
}

// 先随机选一个可用的
func (s *DsService) PickDs() *model.DsModel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ds := range s.dsById {
		if tryOccupy(ds) {
			return ds
		}
	}
	return nil
}

func (s *DsService) ReleaseDs(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateBusy, model.DsStateIdle)
}

func tryOccupy(ds *model.DsModel) bool {
	return ds.State.CompareAndSwap(model.DsStateIdle, model.DsStateBusy)
}
