package protocol

import "gobattle/btrouter"

const (
	// ==================== 通用 ====================
	CodeDBFailed     btrouter.Code = 10001 // 数据库操作失败
	CodeUserNotFound btrouter.Code = 10002 // 玩家不存在

	// ==================== 登录 ====================
	CodeInvalidUsername btrouter.Code = 10101 // username 为空、非法或过长
	CodeAlreadyLogin    btrouter.Code = 10102 // 当前连接已登录
	CodeNotLogin        btrouter.Code = 10103 // 当前连接未登录

	// ==================== DS ====================
	CodeDsInvalidParam      btrouter.Code = 10201 // ds参数错误
	CodeDsAlreadyRegistered btrouter.Code = 10202 // 当前连接已注册
	CodeDsNotRegistered     btrouter.Code = 10203 // 当前连接未注册

	// ==================== 匹配 ====================
	CodeMatchInvalidParam btrouter.Code = 10301 // 匹配参数错误
	CodeMatchAlreadyIn    btrouter.Code = 10302 // 已在匹配中
	CodeMatchNotIn        btrouter.Code = 10303 // 未在匹配中
	CodeMatchNoDs         btrouter.Code = 10304 // 无可用DS服务器

	// ==================== 战斗 ====================
	CodeBattleNotFound   btrouter.Code = 10401 // 战斗不存在或不属于当前DS
	CodeBattleStateError btrouter.Code = 10402 // 战斗状态不对
	CodeBattleAlreadyIn  btrouter.Code = 10403 // 已在战斗中

	// ==================== 道具 ====================
	CodeItemInvalidParam btrouter.Code = 10501 // 道具类型未注册、数量非法或单次条目过多
	CodeItemNotEnough    btrouter.Code = 10502 // 道具不足
)
