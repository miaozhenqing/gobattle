package service

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/game/protocol"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dsStateIdle int32 = 0
	dsStateBusy int32 = 1
)

type Ds struct {
	Id           int64
	IP           string
	Port         int
	RegisterTime int64
	Conn         *btnet.Connection
	State        atomic.Int32
}

var dsKey = btnet.NewAttrKey[*Ds]("service.ds")

func GetDs(conn *btnet.Connection) (*Ds, bool) {
	return dsKey.Get(conn)
}

type DsService struct {
	mu                  sync.RWMutex
	dsById              map[int64]*Ds
	unregisterListeners []func(ds *Ds)
}

func NewDsService() *DsService {
	return &DsService{dsById: make(map[int64]*Ds)}
}

func (s *DsService) OnUnregister(fn func(ds *Ds)) {
	s.unregisterListeners = append(s.unregisterListeners, fn)
}

func (s *DsService) Register(conn *btnet.Connection, dsId int64, ip string, port int) (*Ds, error) {
	if _, ok := dsKey.Get(conn); ok {
		return nil, protocol.CodeDsAlreadyRegistered
	}

	ds := &Ds{
		Id:           dsId,
		IP:           ip,
		Port:         port,
		RegisterTime: time.Now().UnixMilli(),
		Conn:         conn,
	}
	dsKey.Set(conn, ds)

	s.mu.Lock()
	old := s.dsById[ds.Id]
	s.dsById[ds.Id] = ds
	s.mu.Unlock()
	if old != nil {
		btlog.Ins().Info("dsReplaced", "dsId", ds.Id, "oldConnId", old.Conn.GetId(), "newConnId", conn.GetId())
		old.Conn.FlushAndStop()
	}

	btlog.Ins().Info("dsRegister", "dsId", ds.Id, "ip", ds.IP, "port", ds.Port, "connId", conn.GetId())
	return ds, nil
}

func (s *DsService) Unregister(conn *btnet.Connection) {
	ds, ok := dsKey.Get(conn)
	if !ok {
		return
	}
	s.mu.Lock()
	// 只删自己，被顶替的旧连接断开时不能删掉新
	if s.dsById[ds.Id] == ds {
		delete(s.dsById, ds.Id)
	}
	s.mu.Unlock()
	btlog.Ins().Info("dsUnregister", "dsId", ds.Id, "connId", conn.GetId())

	for _, fn := range s.unregisterListeners {
		fn(ds)
	}
}

// 先随机选一个可用的
func (s *DsService) PickDs() *Ds {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ds := range s.dsById {
		if tryOccupy(ds) {
			return ds
		}
	}
	return nil
}

func (s *DsService) ReleaseDs(ds *Ds) bool {
	return ds.State.CompareAndSwap(dsStateBusy, dsStateIdle)
}

func tryOccupy(ds *Ds) bool {
	return ds.State.CompareAndSwap(dsStateIdle, dsStateBusy)
}
