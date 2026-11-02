package repository

import (
	"context"
	"gobattle/game/model"

	"github.com/jmoiron/sqlx"
)

type ResourceRepository struct {
	tx *sqlx.Tx
}

func (r *ResourceRepository) ListByUserId(ctx context.Context, userId int64) ([]*model.UserResource, error) {
	var resources []*model.UserResource
	err := r.tx.SelectContext(ctx, &resources, "select * from `user_item_resource` where userId = ? and count > 0", userId)
	return resources, err
}

func (r *ResourceRepository) ListByItemIds(ctx context.Context, userId int64, itemIds []int) ([]*model.UserResource, error) {
	if len(itemIds) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In("select * from `user_item_resource` where userId = ? and itemId in (?)", userId, itemIds)
	if err != nil {
		return nil, err
	}
	var resources []*model.UserResource
	err = r.tx.SelectContext(ctx, &resources, r.tx.Rebind(query), args...)
	return resources, err
}

// 已存在的行只更新 count、updateTime，id 不变
func (r *ResourceRepository) Upsert(ctx context.Context, resources []*model.UserResource) error {
	if len(resources) == 0 {
		return nil
	}
	_, err := r.tx.NamedExecContext(ctx,
		"insert into `user_item_resource` (id, userId, itemId, count, createTime, updateTime) values (:id, :userId, :itemId, :count, :createTime, :updateTime)"+
			" on duplicate key update count = values(count), updateTime = values(updateTime)",
		resources)
	return err
}
