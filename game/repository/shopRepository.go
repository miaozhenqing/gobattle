package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"gobattle/game/model"
)

type ShopRepository struct {
	tx *sqlx.Tx
}

func (s *ShopRepository) GetByUserIdAndShopIdForUpdate(ctx context.Context, userId int64, shopId int) (*model.UserShop, error) {
	us := &model.UserShop{}
	err := s.tx.GetContext(ctx, us, "select * from user_shop where userId = ? and shopId = ? for update", userId, shopId)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return us, nil
}

func (s *ShopRepository) Insert(ctx context.Context, userShop *model.UserShop) error {
	_, err := s.tx.ExecContext(ctx, "insert into user_shop  values (?, ?, ?, ?, ?, ?, ?)",
		userShop.Id, userShop.UserId, userShop.CreateTime, userShop.UpdateTime, userShop.ShopId, userShop.BuyCount, userShop.LastBuyTime)
	return err
}

func (s *ShopRepository) Update(ctx context.Context, userShop *model.UserShop) error {
	_, err := s.tx.NamedExec("update user_shop set buyCount = ?, lastBuyTime = ?, updateTime = ? where id = ?",
		userShop)
	return err
}
