package repository

import (
	"context"
	"gobattle/game/model"
	"strings"

	"github.com/jmoiron/sqlx"
)

type ItemRepository struct {
	tx *sqlx.Tx
}

// 只返回数量大于 0 且未过期的道具
func (r *ItemRepository) ListByUserId(ctx context.Context, userId int64, now int64) ([]*model.Item, error) {
	var items []*model.Item
	err := r.tx.SelectContext(ctx, &items,
		"SELECT * FROM `item` WHERE userId = ? AND count > 0 AND (expireTime = 0 OR expireTime > ?)",
		userId, now)
	return items, err
}

func (r *ItemRepository) ListByKeys(ctx context.Context, userId int64, keys []model.ItemKey) ([]*model.Item, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	sb.WriteString("SELECT * FROM `item` WHERE userId = ? AND (category, itemId) IN (")
	args := make([]any, 0, 1+len(keys)*2)
	args = append(args, userId)
	for i, key := range keys {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?)")
		args = append(args, key.Category, key.ItemId)
	}
	sb.WriteString(")")
	var items []*model.Item
	err := r.tx.SelectContext(ctx, &items, sb.String(), args...)
	return items, err
}

func (r *ItemRepository) Upsert(ctx context.Context, items []*model.Item) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	//todo 显示写字段？魔法值9？
	sb.WriteString("INSERT INTO `item` (userId, category, itemId, instanceId, count, expireTime, data, createTime, updateTime) VALUES ")
	args := make([]any, 0, len(items)*9)
	for i, item := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args, item.UserId, item.Category, item.ItemId, item.InstanceId, item.Count,
			item.ExpireTime, item.Data, item.CreateTime, item.UpdateTime)
	}
	sb.WriteString(" ON DUPLICATE KEY UPDATE count = VALUES(count), expireTime = VALUES(expireTime), data = VALUES(data), updateTime = VALUES(updateTime)")
	_, err := r.tx.ExecContext(ctx, sb.String(), args...)
	return err
}
