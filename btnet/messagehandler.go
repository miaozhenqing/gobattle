package btnet

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"runtime/debug"
)

type requestTask struct {
	conn btiface.IConnection
	msg  btiface.IMessage
}

type MessageHandler struct {
	workerPoolSize      int
	workerPoolQueueSize int
	workerQueues        []chan requestTask
}

func NewMessageHandler(workerPoolSize, workerPoolQueueSize int) *MessageHandler {
	if workerPoolSize <= 0 {
		panic("workerPoolSize must be > 0")
	}
	m := &MessageHandler{
		workerPoolSize:      workerPoolSize,
		workerPoolQueueSize: workerPoolQueueSize,
		workerQueues:        make([]chan requestTask, workerPoolSize),
	}
	m.StartWorkPool()
	return m
}

func (mh *MessageHandler) HandleMessage(conn btiface.IConnection, msg btiface.IMessage) {
	mh.workerQueues[conn.GetWorkerID()] <- requestTask{conn: conn, msg: msg}
}

func (mh *MessageHandler) doHandleMessage(task requestTask) {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("worker panic, connID:%v err:%v\n%s", task.conn.GetID(), r, debug.Stack())
		}
	}()
	if task.conn.IsClosed() {
		btlog.Ins().Warnf("conn closed, connID:%v", task.conn.GetID())
		return
	}
	p := task.conn.GetPipeline()
	if p == nil {
		btlog.Ins().Errorf("pipeline is nil, connID:%v", task.conn.GetID())
		return
	}
	p.FireRead(task.msg)
}

func (mh *MessageHandler) StartWorkPool() {
	for i := 0; i < mh.workerPoolSize; i++ {
		mh.workerQueues[i] = make(chan requestTask, mh.workerPoolQueueSize)
		go mh.startWorker(i, mh.workerQueues[i])
	}
}

func (mh *MessageHandler) startWorker(i int, taskChan chan requestTask) {
	for {
		select {
		case task, ok := <-taskChan:
			if ok {
				mh.doHandleMessage(task)
			} else {
				btlog.Ins().Warnf("taskChan closed:%v", i)
				return
			}
		}
	}
}

func (mh *MessageHandler) AllocateWorkerID(connID int64) int {
	return int(connID % int64(mh.workerPoolSize))
}
