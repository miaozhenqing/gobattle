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

type Battle struct {
	Id         int64
	GameMode   int
	MapId      int
	Ds         *Ds
	Players    []*BattlePlayer
	State      int
	CreateTime int64
	StartTime  int64
}
