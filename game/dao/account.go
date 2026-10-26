package dao

import (
	"context"
	"gobattle/game/model"

	"github.com/jmoiron/sqlx"
)

func GetAccountByUsernameForUpdate(ctx context.Context, tx *sqlx.Tx, username string) (*model.Account, error) {
	acc := &model.Account{}
	err := tx.GetContext(ctx, acc,
		"SELECT * FROM account WHERE username = ? FOR UPDATE", username)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func InsertAccount(ctx context.Context, ext sqlx.ExtContext, acc *model.Account) (int64, error) {
	res, err := ext.ExecContext(ctx,
		"INSERT INTO account (username, roleName, createTime, lastLoginTime) VALUES (?, ?, ?, ?)",
		acc.Username, acc.RoleName, acc.CreateTime, acc.LastLoginTime)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateAccountLastLoginTime(ctx context.Context, ext sqlx.ExtContext, uid int64, loginTime int64) error {
	_, err := ext.ExecContext(ctx, "UPDATE account SET lastLoginTime = ? WHERE uid = ?", loginTime, uid)
	return err
}
