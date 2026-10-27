package proto

import "gobattle/btrouter"

const (
	// ==================== 通用 ====================
	CodeDBFailed btrouter.Code = 10001 // 数据库操作失败

	// ==================== 登录 ====================
	CodeInvalidUsername btrouter.Code = 10101 // username 为空、非法或过长
	CodeAlreadyLogin    btrouter.Code = 10102 // 当前连接已登录
	CodeNotLogin        btrouter.Code = 10103 // 当前连接未登录

	// ==================== DS ====================
	CodeDsInvalidParam      btrouter.Code = 10201 // ds参数错误
	CodeDsAlreadyRegistered btrouter.Code = 10202 // 当前连接已注册
	CodeDsNotRegistered     btrouter.Code = 10203 // 当前连接未注册
)
