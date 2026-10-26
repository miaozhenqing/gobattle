package proto

const (
	// ==================== 登录 ====================
	MsgLogin uint32 = 1101
	// ==================== 推送 ====================
	PushKick uint32 = 51002
)

type LoginReq struct {
	Username string `json:"username"` // 建号时同时作为角色名
}

type LoginResp struct {
	UID      int64  `json:"uid"`
	Username string `json:"username"`
	RoleName string `json:"roleName"`
	IsNew    bool   `json:"isNew"` // 本次新建账号
}

// KickPush 账号在别处登录
type KickPush struct{}
