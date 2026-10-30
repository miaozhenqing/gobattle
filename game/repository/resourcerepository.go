package repository

import (
	"context"
	"gobattle/game/model"
	"strings"

	"github.com/jmoiron/sqlx"
)

type ResourceRepository struct {
	tx *sqlx.Tx
}

func (r *ResourceRepository) ListByUserId(ctx context.Context, userId int64) ([]*model.Resource, error) {
	var resources []*model.Resource
	err := r.tx.SelectContext(ctx, &resources, "SELECT * FROM `item_resource` WHERE userId = ? AND count > 0", userId)
	return resources, err
}

func (r *ResourceRepository) ListByItemIds(ctx context.Context, userId int64, itemIds []int) ([]*model.Resource, error) {
	if len(itemIds) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In("SELECT * FROM `item_resource` WHERE userId = ? AND itemId IN (?)", userId, itemIds)
	if err != nil {
		return nil, err
	}
	var resources []*model.Resource
	err = r.tx.SelectContext(ctx, &resources, r.tx.Rebind(query), args...)
	return resources, err
}

// 已存在的行只更新 count、updateTime，id 不变
func (r *ResourceRepository) Upsert(ctx context.Context, resources []*model.Resource) error {
	if len(resources) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO `item_resource` (id, userId, itemId, count, createTime, updateTime) VALUES ")
	args := make([]any, 0, len(resources)*6)
	for i, res := range resources {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?)")
		args = append(args, res.Id, res.UserId, res.ItemId, res.Count, res.CreateTime, res.UpdateTime)
	}
	sb.WriteString(" ON DUPLICATE KEY UPDATE count = VALUES(count), updateTime = VALUES(updateTime)")
	_, err := r.tx.ExecContext(ctx, sb.String(), args...)
	return err
}
