package proto

const (
	// ==================== 登录 ====================
	MsgLogin     uint32 = 1101
	MsgHeartBeat uint32 = 1102
	// ==================== DS ====================
	MsgDsRegister  uint32 = 1201
	MsgDsHeartBeat uint32 = 1202
	MsgDsRelease   uint32 = 1203
	// ==================== 匹配 ====================
	MsgMatchStart  uint32 = 1301
	MsgMatchCancel uint32 = 1302
	// ==================== 局内 ====================
	MsgDsBattleEnd uint32 = 1401
	// ==================== 推送 ====================
	PushKick         uint32 = 51002
	PushMatchSuccess uint32 = 51003
	PushMatchFail    uint32 = 51004
)

// ==============user
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

// ==============ds
type DsReleaseReq struct {
	BattleId int64 `json:"battleId"`
}

type DsReleaseResp struct {
	DsID int64 `json:"dsId"`
}

type DsBattleEndReq struct {
	BattleId int64 `json:"battleId"`
	GameMode int   `json:"gameMode"`
	//统计项id->value
	Statistics map[int]int `json:"statistics"`
}

type DsBattleEndResp struct {
	BattleId int64 `json:"battleId"`
}

// ==============ds
// ==============match
type MatchStartReq struct {
	GameMode int `json:"gameMode"`
	MapID    int `json:"mapId"`
}

type MatchStartResp struct {
	GameMode int `json:"gameMode"`
	MapID    int `json:"mapId"`
}

type MatchCancelReq struct{}

type MatchCancelResp struct{}

type MatchPlayerInfo struct {
	UID        int64 `json:"uid"`
	MatchScore int   `json:"matchScore"`
	Team       int   `json:"team"`
}

type MatchSuccessPush struct {
	BattleId int64              `json:"battleId"`
	GameMode int                `json:"gameMode"`
	MapID    int                `json:"mapId"`
	DsIP     string             `json:"dsIp"`
	DsPort   int                `json:"dsPort"`
	Players  []*MatchPlayerInfo `json:"players"`
}

type MatchFailPush struct {
	Code uint32 `json:"code"`
}
