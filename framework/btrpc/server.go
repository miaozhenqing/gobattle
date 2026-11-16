package btrpc

import (
	"context"
	"errors"
	"fmt"
	"gobattle/framework/btconfig"
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

var (
	_ btnet.InHandler     = (*Server)(nil)
	_ btnet.ActiveHandler = (*Server)(nil)

	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()

	gatewayIdKey = btnet.NewAttrKey[int64]("btrpc.gatewayId")
)

const slowCallThreshold = 50 * time.Millisecond

type method struct {
	name    string
	reqType reflect.Type
	fn      reflect.Value
}

type Server struct {
	codec   btrouter.Codec
	group   *btnet.ServerGroup
	biz     *btnet.EventLoopGroup
	methods map[string]*method
	frozen  atomic.Bool

	gwMu     sync.RWMutex
	gateways map[int64]*btnet.Connection
}

func NewServer(cfg btconfig.ServerConfig, codec btrouter.Codec) *Server {
	s := &Server{codec: codec, methods: make(map[string]*method), gateways: make(map[int64]*btnet.Connection)}
	boss := btnet.NewEventLoopGroup("rpc-boss", 1)
	worker := btnet.NewEventLoopGroup("rpc-worker", cfg.IOLoopSize)
	s.biz = btnet.NewEventLoopGroup("rpc-biz", cfg.BizLoopSize)
	srv := btnet.NewServer(cfg.Options(), boss, worker)
	srv.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(PacketCodec{})
		p.AddLast(s)
	})
	s.group = &btnet.ServerGroup{Name: "rpc", Server: srv, Groups: []*btnet.EventLoopGroup{boss, worker, s.biz}}
	return s
}

func Register[T any](s *Server, impl T) {
	it := reflect.TypeOf((*T)(nil)).Elem()
	if it.Kind() != reflect.Interface {
		panic(fmt.Sprintf("btrpc register:%v is not interface", it))
	}
	if s.frozen.Load() {
		panic(fmt.Sprintf("btrpc register frozen"))
	}
	v := reflect.ValueOf(impl)
	for i := range it.NumMethod() {
		m := it.Method(i)
		t := m.Type
		//todo 方法签名校验

		rpcName := it.Name() + "." + m.Name
		if _, ok := s.methods[rpcName]; ok {
			panic(fmt.Sprintf("btrpc register duplicate") + rpcName)
		}
		s.methods[rpcName] = &method{
			name:    rpcName,
			reqType: t.In(1).Elem(),
			fn:      v.MethodByName(m.Name),
		}
	}
}

func (s *Server) Start() error {
	s.frozen.Store(true)
	return s.group.Server.Start()
}

func (s *Server) Shutdown(ctx context.Context) {
	s.group.Shutdown(ctx)
}

func (s *Server) OnRead(ctx *btnet.HandlerContext, msg any) {

}

func (s *Server) OnActive(ctx *btnet.HandlerContext) { panic("TODO: activate RPC connection") }

func (s *Server) OnInactive(ctx *btnet.HandlerContext) { panic("TODO: unbind RPC gateway") }

func (s *Server) bindGatewayConnection(conn *btnet.Connection, gatewayId int64) {
	panic("TODO: bind RPC gateway")
}

func (s *Server) getGatewayConnection(gatewayId int64) (*btnet.Connection, bool) {
	panic("TODO: find RPC gateway")
}

func (s *Server) dispatchRequest(conn *btnet.Connection, p *Packet) {
	panic("TODO: dispatch RPC request")
}

func (s *Server) callMethod(ctx context.Context, m *method, payload []byte) (any, error) {
	return nil, errors.New("TODO: invoke RPC method")
}

func (s *Server) sendResponse(conn *btnet.Connection, req *Packet, resp any, err error) {
	panic("TODO: reply to RPC request")
}

func (s *Server) rpcErrorCode(req *Packet, err error) btrouter.Code {
	panic("TODO: map RPC error code")
}

func (s *Server) PushCtx(ctx context.Context, msgId uint32, v any) error {
	return errors.New("TODO: push RPC context message")
}

func (s *Server) Push(gatewayId, userId int64, msgId uint32, v any) error {
	return errors.New("TODO: push RPC message")
}

func (s *Server) Broadcast(msgId uint32, v any) error {
	return errors.New("TODO: broadcast RPC message")
}
