package main

import (
	"context"
	"flag"
	"fmt"
	"gobattle/common/protocol"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9528", "gateway addr")
	username := flag.String("username", "test1", "login username")
	flag.Parse()

	client := dial(*addr)
	defer client.Close()

	loginResp, err := call[protocol.LoginResp](client, protocol.MsgLogin, &protocol.LoginReq{Username: *username})
	btlog.Ins().Info("MsgLogin", "resp", loginResp, "err", err)

	gmResp, err := call[protocol.GMResp](client, protocol.MsgGM, &protocol.GMReq{
		Name: "GMItemAdd",
		Args: fmt.Sprintf("%d|%d|%d|%d", loginResp.UserId, 1, 10001, 100),
	})
	btlog.Ins().Info("MsgLogin", "resp", gmResp, "err", err)

	itemListResp, err := call[protocol.ItemListResp](client, protocol.MsgItemList, &protocol.ItemListReq{})
	if err != nil {
		btlog.Ins().Error("MsgItemList", "err", err)
	} else {
		for _, item := range itemListResp.Items {
			btlog.Ins().Info("itemList", "id", item.Id, "category", item.Category, "itemId", item.ItemId, "count", item.Count)
		}
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
