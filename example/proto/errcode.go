package proto

import "gobattle/btrouter"

const (
	CodeEmptyText btrouter.Code = 10001 // text 为空

	CodeLoginFailed  btrouter.Code = 10101 // token 无效
	CodeAlreadyLogin btrouter.Code = 10102 // 当前连接已登录
)
