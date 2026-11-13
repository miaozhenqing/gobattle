package btrpc

import (
	"context"
	"errors"
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
	panic("TODO: construct RPC server")
}

func Register[T any](s *Server, impl T) { panic("TODO: register RPC interface methods") }

func (s *Server) Start() error { return errors.New("TODO: start RPC server") }

func (s *Server) Shutdown(ctx context.Context) { panic("TODO: shut down RPC server") }

func (s *Server) OnRead(ctx *btnet.HandlerContext, msg any) { panic("TODO: dispatch RPC packet") }

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
