package model

const (
	BattleStateWaiting = 1 // 已分配ds
	BattleStateRunning = 2 // 战斗中
)

type BattlePlayer struct {
	UserId     int64
	MatchScore int
	Team       int
}

type BattleModel struct {
	ID         int64
	GameMode   int
	MapID      int
	Ds         *DsModel
	Players    []*BattlePlayer
	State      int
	CreateTime int64
	StartTime  int64
}
