package proto

const (
	// ==================== 登录 ====================
	MsgLogin     uint32 = 1101
	MsgHeartBeat uint32 = 1102
	// ==================== DS ====================
	MsgDsRegister  uint32 = 1201
	MsgDsHeartBeat uint32 = 1202
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

type KickPush struct{}

type HeartBeatReq struct {
	UID int64 `json:"uid"`
}

type HeartBeatResp struct {
	UID int64 `json:"uid"`
}

type DsRegisterReq struct {
	DsID int64  `json:"dsId"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type DsRegisterResp struct {
	DsID int64 `json:"dsId"`
}

type DsHeartBeatReq struct{}

type DsHeartBeatResp struct {
	DsID int64 `json:"dsId"`
}
