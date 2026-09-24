package btnet

import (
	"fmt"
	"gobattle/btiface"
)

type MessageHandler struct {
	serializer btiface.ISerializer
	apis       map[uint32]btiface.IRouter
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		apis: make(map[uint32]btiface.IRouter),
	}
}

func (mh *MessageHandler) SetSerializer(s btiface.ISerializer) {
	mh.serializer = s
}

func (mh *MessageHandler) AddRouter(msgId uint32, r btiface.IRouter) {
	mh.apis[msgId] = r
}

func (mh *MessageHandler) HandleMessage(request btiface.IRequest) {
	var data = request.GetData()
	// 反序列化
	msg, _ := mh.serializer.Deserialize(data)

	// 处理消息
	router := mh.apis[msg.GetMsgId()]
	if router == nil {
		fmt.Printf("error msgId:%s", msg.GetMsgId())
		return
	}

	request.HandleRouter(router)

}
