package protocol

const (
	// ==================== 登录 ====================
	MsgLogin     uint32 = 1101
	MsgHeartBeat uint32 = 1102
	// ==================== DS ====================
	MsgDsRegister  uint32 = 1201
	MsgDsHeartBeat uint32 = 1202
	// ==================== 匹配 ====================
	MsgMatchStart  uint32 = 1301
	MsgMatchCancel uint32 = 1302
	// ==================== 局内 ====================
	MsgDsBattleEnd   uint32 = 1401
	MsgDsBattleStart uint32 = 1402
	MsgBattleQuery   uint32 = 1403
	// ==================== 道具 ====================
	MsgItemList uint32 = 1501
	// ==================== 推送 ====================
	PushKick         uint32 = 51002
	PushMatchSuccess uint32 = 51003
	PushMatchFail    uint32 = 51004
	PushBattleEnd    uint32 = 51005
	PushItemChange   uint32 = 51006
	// ==================== 推送给DS ====================
	PushDsBattleAssign uint32 = 51101
)

// 战斗结束原因
const (
	BattleEndNormal = 1
	BattleEndAbort  = 2
)

// ==============user
type LoginReq struct {
	Username string `json:"username"`
}

type LoginResp struct {
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	RoleName string `json:"roleName"`
	IsNew    bool   `json:"isNew"`
}

type KickPush struct{}

type HeartBeatReq struct {
	UserId int64 `json:"userId"`
}

type HeartBeatResp struct {
	UserId int64 `json:"userId"`
}

// ==============ds
type DsRegisterReq struct {
	DsId int64  `json:"dsId"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type DsRegisterResp struct {
	DsId int64 `json:"dsId"`
}

type DsHeartBeatReq struct{}

type DsHeartBeatResp struct {
	DsId int64 `json:"dsId"`
}

// ==============battle
// DsBattleAssignPush 匹配成功后通知DS本局信息，DS按名单等玩家进房
type DsBattleAssignPush struct {
	BattleId int64              `json:"battleId"`
	GameMode int                `json:"gameMode"`
	MapId    int                `json:"mapId"`
	Players  []*MatchPlayerInfo `json:"players"`
}

type DsBattleStartReq struct {
	BattleId int64 `json:"battleId"`
}

type DsBattleStartResp struct {
	BattleId int64 `json:"battleId"`
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

type BattleQueryReq struct{}

// BattleQueryResp 不在战斗中时 BattleId 为 0
type BattleQueryResp struct {
	BattleId int64  `json:"battleId"`
	GameMode int    `json:"gameMode"`
	MapId    int    `json:"mapId"`
	DsIP     string `json:"dsIp"`
	DsPort   int    `json:"dsPort"`
	State    int    `json:"state"`
}

type BattleEndPush struct {
	BattleId int64 `json:"battleId"`
	Reason   int   `json:"reason"`
}

// ==============match
type MatchStartReq struct {
	GameMode int `json:"gameMode"`
	MapId    int `json:"mapId"`
}

type MatchStartResp struct {
	GameMode int `json:"gameMode"`
	MapId    int `json:"mapId"`
}

type MatchCancelReq struct{}

type MatchCancelResp struct{}

type MatchPlayerInfo struct {
	UserId     int64 `json:"userId"`
	MatchScore int   `json:"matchScore"`
	Team       int   `json:"team"`
}

type MatchSuccessPush struct {
	BattleId int64              `json:"battleId"`
	GameMode int                `json:"gameMode"`
	MapId    int                `json:"mapId"`
	DsIP     string             `json:"dsIp"`
	DsPort   int                `json:"dsPort"`
	Players  []*MatchPlayerInfo `json:"players"`
}

type MatchFailPush struct {
	Code uint32 `json:"code"`
}

// ==============item

type ItemInfo struct {
	Id       int64 `json:"id"`
	Category int   `json:"category"`
	ItemId   int   `json:"itemId"`
	Count    int64 `json:"count"`
}

type ItemListReq struct{}

type ItemListResp struct {
	Items []*ItemInfo `json:"items"`
}

type ItemChangeInfo struct {
	ItemInfo
	Before int64 `json:"before"`
}

type ItemChangePush struct {
	Reason  int               `json:"reason"`
	Changes []*ItemChangeInfo `json:"changes"`
}
