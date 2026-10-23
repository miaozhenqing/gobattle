package proto

const (
	// ==================== ping ====================
	MsgPing uint32 = 1001
	// ==================== 广播 ====================
	PushBroadcast uint32 = 51001
)

type PingReq struct {
	Text string `json:"text"`
}

type PingResp struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
}
