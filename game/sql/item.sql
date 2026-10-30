CREATE TABLE IF NOT EXISTS `item_resource` (
  `id`         BIGINT NOT NULL COMMENT 'id',
  `userId`     BIGINT NOT NULL DEFAULT 0 COMMENT '玩家id',
  `itemId`     INT    NOT NULL DEFAULT 0 COMMENT '道具id',
  `count`      BIGINT NOT NULL DEFAULT 0 COMMENT '数量',
  `createTime` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updateTime` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_userId_itemId` (`userId`, `itemId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='玩家资源（金币等货币）';

CREATE TABLE IF NOT EXISTS `item_weapon` (
  `id`         BIGINT NOT NULL COMMENT 'id',
  `userId`     BIGINT NOT NULL DEFAULT 0 COMMENT '玩家id',
  `itemId`     INT    NOT NULL DEFAULT 0 COMMENT '道具id',
  `count`      BIGINT NOT NULL DEFAULT 0 COMMENT '数量',
  `level`      INT    NOT NULL DEFAULT 1 COMMENT '武器等级',
  `createTime` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间',
  `updateTime` BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_userId_itemId` (`userId`, `itemId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='玩家武器';

