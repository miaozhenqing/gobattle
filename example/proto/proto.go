package proto

const (
	// ==================== ping ====================
	MsgPing uint32 = 1001
	// ==================== 登录 ====================
	MsgLogin uint32 = 1101
	// ==================== 广播 ====================
	PushBroadcast uint32 = 51001
	PushKick      uint32 = 51002
)

type PingReq struct {
	Text string `json:"text"`
}

type PingResp struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
}

type LoginReq struct {
	Token string `json:"token"`
}

type LoginResp struct {
	UID int64 `json:"uid"`
}

// KickPush 账号在别处登录
type KickPush struct{}
