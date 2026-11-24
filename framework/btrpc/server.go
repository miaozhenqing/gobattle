package btrpc

import (
	"context"
	"errors"
	"fmt"
	"gobattle/framework/btconfig"
	"gobattle/framework/btlog"
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var (
	_ btnet.InHandler     = (*Server)(nil)
	_ btnet.ActiveHandler = (*Server)(nil)

	ctxType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errType = reflect.TypeOf((*error)(nil)).Elem()

	peerKey = btnet.NewAttrKey[Peer]("btrpc.peer")
)

type Peer struct {
	Type string
	Id   int64
}

type rpcConnectionKey struct{}

type method struct {
	name         string       //格式："UserApi.Login"
	reqType      reflect.Type //Req 结构体类型（非指针）
	fn           reflect.Value
	allowUnbound bool
}

type Server struct {
	codec   btrouter.Codec
	group   *btnet.ServerGroup
	biz     *btnet.EventLoopGroup
	methods map[string]*method
	frozen  atomic.Bool

	mu    sync.RWMutex
	peers map[Peer]*btnet.Connection
}

func NewServer(cfg btconfig.ServerConfig, codec btrouter.Codec) *Server {
	if codec == nil {
		codec = btrouter.JSONCodec{}
	}
	s := &Server{codec: codec, methods: make(map[string]*method), peers: make(map[Peer]*btnet.Connection)}
	worker := btnet.NewEventLoopGroup("rpc-worker", cfg.IOLoops())
	s.biz = btnet.NewEventLoopGroup("rpc-biz", cfg.BizLoops())
	srv := btnet.NewServer(cfg.Options(), worker)
	srv.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(PacketCodec{})
		p.AddLast(s)
	})
	s.group = &btnet.ServerGroup{Name: "rpc", Server: srv, Groups: []*btnet.EventLoopGroup{worker, s.biz}}
	return s
}

// func(context.Context, *Req) (*Resp, error)
func Register[T any](s *Server, impl T) {
	registerMethods(s, impl, "")
}

func RegisterBeforeBind[T any](s *Server, impl T, methodName string) {
	registerMethods(s, impl, methodName)
}

func registerMethods[T any](s *Server, impl T, beforeBindMethod string) {
	it := reflect.TypeOf((*T)(nil)).Elem()
	if it.Kind() != reflect.Interface {
		panic(fmt.Sprintf("btrpc register:%v is not interface", it))
	}
	if s.frozen.Load() {
		panic(fmt.Sprintf("btrpc register frozen"))
	}
	v := reflect.ValueOf(impl)
	if beforeBindMethod != "" {
		if _, ok := it.MethodByName(beforeBindMethod); !ok {
			panic("btrpc register missing before-bind method " + beforeBindMethod)
		}
	}
	for i := range it.NumMethod() {
		m := it.Method(i)
		t := m.Type
		//todo 方法签名校验

		rpcName := it.Name() + "." + m.Name
		if _, ok := s.methods[rpcName]; ok {
			panic("btrpc register duplicate " + rpcName)
		}
		s.methods[rpcName] = &method{
			name:         rpcName,
			reqType:      t.In(1).Elem(),
			fn:           v.MethodByName(m.Name),
			allowUnbound: m.Name == beforeBindMethod,
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
	p, ok := msg.(*Packet)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	conn := ctx.GetConnection()
	if p.Type != btnet.MessageRequest {
		btlog.Ins().Warn("rpcUnexpectedPacket", "connId", conn.GetId(), "type", p.Type)
		return
	}
	m := s.methods[p.Method]
	if m == nil {
		btlog.Ins().Warn("rpcMethodNotFound", "connId", conn.GetId(), "rpcMethod", p.Method, "seq", p.Seq)
		s.sendResponse(conn, p, nil, btrouter.CodeUnimplemented)
		return
	}
	if _, bound := peerKey.Get(conn); !bound && !m.allowUnbound {
		btlog.Ins().Warn("rpcPeerNotBound", "connId", conn.GetId(), "rpcMethod", p.Method, "seq", p.Seq)
		s.sendResponse(conn, p, nil, btrouter.CodeUnauthenticated)
		return
	}
	key := p.UserId
	if key == 0 {
		key = conn.GetId()
	}
	s.biz.Next(key).Execute(func() {
		s.dispatchRequest(conn, p, m)
	})
}

func (s *Server) OnActive(ctx *btnet.HandlerContext) {
	ctx.FireActive()
}

func (s *Server) OnInactive(ctx *btnet.HandlerContext) {
	conn := ctx.GetConnection()
	peer, ok := peerKey.Get(conn)
	if ok {
		s.mu.Lock()
		if s.peers[peer] == conn {
			delete(s.peers, peer)
		}
		s.mu.Unlock()
	}
	btlog.Ins().Info("rpcConnInactive", "connId", conn.GetId(), "peerType", peer.Type, "peerId", peer.Id)
	ctx.FireInactive()
}

func (s *Server) BindPeer(ctx context.Context, peer Peer) error {
	conn, _ := ctx.Value(rpcConnectionKey{}).(*btnet.Connection)
	if conn == nil || peer.Type == "" || peer.Id <= 0 {
		btlog.Ins().Warn("rpcBindPeerInvalid", "peerType", peer.Type, "peerId", peer.Id)
		return btrouter.CodeInvalidArgument
	}
	s.mu.Lock()
	if bound, ok := peerKey.Get(conn); ok {
		s.mu.Unlock()
		btlog.Ins().Warn("rpcBindPeerAlreadyBound", "connId", conn.GetId(), "peerType", peer.Type, "peerId", peer.Id,
			"boundPeerType", bound.Type, "boundPeerId", bound.Id)
		return btrouter.CodeInvalidArgument
	}
	old := s.peers[peer]
	s.peers[peer] = conn
	peerKey.Set(conn, peer)
	s.mu.Unlock()
	btlog.Ins().Info("rpcPeerBound", "connId", conn.GetId(), "peerType", peer.Type, "peerId", peer.Id)
	if old != nil && old != conn {
		btlog.Ins().Warn("rpcPeerReplaced", "peerType", peer.Type, "peerId", peer.Id, "oldConnId", old.GetId(), "newConnId", conn.GetId())
		old.Stop()
	}
	return nil
}

func (s *Server) getPeerConnection(peer Peer) (*btnet.Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.peers[peer]
	return conn, ok
}

func (s *Server) dispatchRequest(conn *btnet.Connection, p *Packet, m *method) {
	peer, _ := peerKey.Get(conn)
	ctx := WithRequestMeta(conn.Context(), RequestMeta{
		UserId:   p.UserId,
		PeerType: peer.Type,
		PeerId:   peer.Id,
	})
	ctx = context.WithValue(ctx, rpcConnectionKey{}, conn)
	startAt := time.Now()
	resp, err := s.callMethod(ctx, m, p.Payload)
	btlog.Ins().Debug("rpcRequest", "connId", conn.GetId(), "rpcMethod", m.name, "seq", p.Seq, "userId", p.UserId,
		"peerType", peer.Type, "peerId", peer.Id, "req", string(p.Payload), "resp", resp, "err", err,
		"durationMs", time.Since(startAt).Milliseconds())
	s.sendResponse(conn, p, resp, err)
}

func (s *Server) callMethod(ctx context.Context, m *method, payload []byte) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Error("rpcPanic", "rpcMethod", m.name, "userId", UserId(ctx), "panic", r, "stack", string(debug.Stack()))
			resp, err = nil, btrouter.CodeInternal
		}
		//todo 可以在此处埋点耗时等
	}()
	req := reflect.New(m.reqType)
	if len(payload) > 0 {
		if err := s.codec.Unmarshal(payload, req.Interface()); err != nil {
			btlog.Ins().Warn("rpcDecodeError", "rpcMethod", m.name, "err", err)
			return nil, btrouter.CodeInvalidArgument
		}
	}
	out := m.fn.Call([]reflect.Value{reflect.ValueOf(ctx), req})
	err, _ = out[1].Interface().(error)
	if out[0].IsNil() {
		return nil, err
	}
	return out[0].Interface(), err
}

func (s *Server) sendResponse(conn *btnet.Connection, req *Packet, resp any, err error) {
	code := s.rpcErrorCode(req, err)
	if req.Seq == 0 {
		//无需回包
		return
	}
	out := &Packet{Type: btnet.MessageResponse, Seq: req.Seq, Code: uint32(code)}
	if resp != nil {
		b, err := s.codec.Marshal(resp)
		if err != nil {
			btlog.Ins().Error("rpcMarshalRespError", "rpcMethod", req.Method, "err", err)
			out.Code = uint32(btrouter.CodeInternal)
		} else {
			out.Payload = b
		}
	}
	conn.Write(out)
}

func (s *Server) rpcErrorCode(req *Packet, err error) btrouter.Code {
	if err == nil {
		return btrouter.CodeOK
	}
	var code btrouter.Code
	if errors.As(err, &code) {
		if !errors.Is(err, error(code)) {
			btlog.Ins().Warn("rpcCodeError", "rpcMethod", req.Method, "userId", req.UserId, "code", uint32(code), "err", err)
		}
		return code
	}
	btlog.Ins().Error("rpcNonCodeError", "rpcMethod", req.Method, "userId", req.UserId, "err", err)
	return btrouter.CodeInternal
}

func (s *Server) PushCtx(ctx context.Context, msgId uint32, v any) error {
	m := GetRequestMeta(ctx)
	return s.Push(Peer{
		Type: m.PeerType,
		Id:   m.PeerId,
	}, m.UserId, msgId, v)
}

func (s *Server) Push(peer Peer, userId int64, msgId uint32, v any) error {
	conn, ok := s.getPeerConnection(peer)
	if !ok {
		btlog.Ins().Warn("rpcPushPeerNotFound", "peerType", peer.Type, "peerId", peer.Id, "userId", userId, "msgId", msgId)
		return btrouter.CodeUnavailable
	}
	data, err := s.codec.Marshal(v)
	if err != nil {
		return err
	}
	btlog.Ins().Debug("rpcPush", "connId", conn.GetId(), "peerType", peer.Type, "peerId", peer.Id, "userId", userId, "msgId", msgId, "data", v)
	return conn.Write(&Packet{Type: btnet.MessagePush, MsgId: msgId, UserId: userId, Payload: data})
}

func (s *Server) Broadcast(peerType string, userId int64, msgId uint32, v any) error {
	data, err := s.codec.Marshal(v)
	if err != nil {
		return err
	}
	//编码一次
	frame := encodePacket(&Packet{Type: btnet.MessagePush, MsgId: msgId, UserId: userId, Payload: data})
	s.mu.RLock()
	conns := make([]*btnet.Connection, 0, len(s.peers))
	for peer, c := range s.peers {
		if peer.Type == peerType {
			conns = append(conns, c)
		}
	}
	s.mu.RUnlock()
	btlog.Ins().Debug("rpcBroadcast", "peerType", peerType, "userId", userId, "conns", len(conns), "msgId", msgId, "data", v)
	for _, c := range conns {
		c.Write(frame)
	}
	return nil
}
