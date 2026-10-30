package model

type User struct {
	UserId        int64  `db:"id"`
	Username      string `db:"username"`
	RoleName      string `db:"roleName"`
	CreateTime    int64  `db:"createTime"`
	LastLoginTime int64  `db:"lastLoginTime"`
	UpdateTime    int64  `db:"updateTime"`
}
