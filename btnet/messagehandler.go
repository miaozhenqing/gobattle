package btnet

import (
	"fmt"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/config"
)

type MessageHandler struct {
	serializer btiface.ISerializer
	apis       map[uint32]btiface.IRouter

	workerPoolSize      uint32
	workerPoolQueueSize uint32
	workerQueues        []chan btiface.IRequest
}

func NewMessageHandler() *MessageHandler {
	m := &MessageHandler{
		apis:                make(map[uint32]btiface.IRouter),
		workerPoolSize:      config.ServerConfig.WorkerPoolSize,
		workerPoolQueueSize: config.ServerConfig.WorkerPoolQueueSize,
		workerQueues:        make([]chan btiface.IRequest, config.ServerConfig.WorkerPoolSize),
	}
	m.StartWorkPool()
	return m
}

func (mh *MessageHandler) SetSerializer(s btiface.ISerializer) {
	mh.serializer = s
}

func (mh *MessageHandler) AddRouter(msgId uint32, r btiface.IRouter) {
	mh.apis[msgId] = r
}

func (mh *MessageHandler) HandleMessage(request btiface.IRequest) {

	workerQueue := mh.workerQueues[request.GetConnection().GetWorkerID()]
	workerQueue <- request
}

func (mh *MessageHandler) doHandleMessage(request btiface.IRequest) {
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

	PutRequest(request)
}

func (mh *MessageHandler) StartWorkPool() {
	for i := 0; i < int(mh.workerPoolSize); i++ {
		mh.workerQueues[i] = make(chan btiface.IRequest, mh.workerPoolQueueSize)
		go mh.startWorker(i, mh.workerQueues[i])
	}
}

func (mh *MessageHandler) startWorker(i int, requestChan chan btiface.IRequest) {
	for {
		select {
		case req, ok := <-requestChan:
			if ok {
				mh.doHandleMessage(req)
			} else {
				btlog.Ins().Warnf("requestChan closed:%v", i)
				return
			}
		}
	}
}

func (mh *MessageHandler) AllocateWorkerID(connID int64) int {
	return int(connID % int64(mh.workerPoolSize))
}
