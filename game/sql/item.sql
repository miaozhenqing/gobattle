CREATE TABLE IF NOT EXISTS `item` (
  `id`         BIGINT NOT NULL AUTO_INCREMENT,
  `userId`     BIGINT NOT NULL COMMENT '玩家 id，关联 user.id',
  `category`   INT    NOT NULL COMMENT '道具类型',
  `itemId`     INT    NOT NULL COMMENT '道具配置 id',
  `instanceId` BIGINT NOT NULL DEFAULT 0 COMMENT '0 为合并行；>0 为批次/独立行，玩家内唯一，取自 user.itemSeq',
  `count`      BIGINT NOT NULL COMMENT '数量；合并行扣到 0 保留，批次/独立行扣完删除',
  `expireTime` BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间，毫秒时间戳，0 为永久',
  `data`       JSON   NULL     COMMENT '专属数据，结构由 category 决定',
  `createTime` BIGINT NOT NULL COMMENT '获得时间，毫秒时间戳',
  `updateTime` BIGINT NOT NULL COMMENT '最近变动时间，毫秒时间戳',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_userId_category_itemId_instanceId` (`userId`, `category`, `itemId`, `instanceId`),
  KEY `idx_userId_itemId` (`userId`, `itemId`),
  KEY `idx_expireTime` (`expireTime`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='玩家道具';

-- 已有 item_gold 的库手动执行：
-- RENAME TABLE `item_gold` TO `item`;
-- ALTER TABLE `item`
--   DROP INDEX `uk_userId_itemId`,
--   ADD COLUMN `instanceId` BIGINT NOT NULL DEFAULT 0 COMMENT '0 为合并行；>0 为批次/独立行，玩家内唯一，取自 user.itemSeq' AFTER `itemId`,
--   ADD COLUMN `expireTime` BIGINT NOT NULL DEFAULT 0 COMMENT '过期时间，毫秒时间戳，0 为永久' AFTER `count`,
--   ADD COLUMN `data` JSON NULL COMMENT '专属数据，结构由 category 决定' AFTER `expireTime`,
--   ADD UNIQUE KEY `uk_userId_category_itemId_instanceId` (`userId`, `category`, `itemId`, `instanceId`),
--   ADD KEY `idx_userId_itemId` (`userId`, `itemId`),
--   ADD KEY `idx_expireTime` (`expireTime`),
--   COMMENT='玩家道具';
