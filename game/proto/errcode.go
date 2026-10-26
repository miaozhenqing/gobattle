package proto

import "gobattle/btrouter"

const (
	// ==================== 通用 ====================
	CodeDBFailed btrouter.Code = 10001 // 数据库操作失败

	// ==================== 登录 ====================
	CodeInvalidUsername btrouter.Code = 10101 // username 为空、非法或过长
	CodeAlreadyLogin    btrouter.Code = 10102 // 当前连接已登录
)
