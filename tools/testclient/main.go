package main

import (
	"context"
	"flag"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/shared/protocol"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9528", "gateway addr")
	username := flag.String("username", "test1", "login username")
	flag.Parse()

	client := dial(*addr)
	defer client.Close()

	loginResp, err := call[protocol.LoginResp](client, protocol.MsgLogin, &protocol.LoginReq{Username: *username})
	if err != nil {
		btlog.Ins().Error("MsgLogin", "err", err)
		return
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_, _ = call[protocol.HeartBeatResp](client, protocol.MsgHeartBeat, &protocol.HeartBeatReq{UserId: loginResp.UserId})
			case <-client.Done():
				return
			}
		}
	}()
	<-client.Done()
	btlog.Ins().Info("clientExit")
}

func dial(addr string) *btrouter.Client {
	return dialWithPush(addr, onPush)
}

func dialWithPush(addr string, onPush func(msgId uint32, data []byte)) *btrouter.Client {
	client, err := btrouter.Dial(addr, 4096, btrouter.JSONCodec{}, onPush)
	if err != nil {
		panic(err)
	}
	return client
}

func onPush(msgId uint32, data []byte) {
	btlog.Ins().Info("clientPush", "msgId", msgId, "data", string(data))
}

func call[Resp any](client *btrouter.Client, msgId uint32, req any) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour) //方便测试debug
	defer cancel()
	return btrouter.Call[Resp](ctx, client, msgId, req)
}
