package model

type Account struct {
	UID           int64  `db:"uid"`
	Username      string `db:"username"`
	RoleName      string `db:"roleName"`      // 建号时取 username
	CreateTime    int64  `db:"createTime"`    // 毫秒时间戳
	LastLoginTime int64  `db:"lastLoginTime"` // 毫秒时间戳
}
