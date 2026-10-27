package model

type UserModel struct {
	UID           int64  `db:"uid"`
	Username      string `db:"username"`
	RoleName      string `db:"roleName"`
	CreateTime    int64  `db:"createTime"`
	LastLoginTime int64  `db:"lastLoginTime"`
}
