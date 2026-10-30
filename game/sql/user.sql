CREATE TABLE IF NOT EXISTS `user` (
  `id`            BIGINT      NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(32) NOT NULL COMMENT '登录名',
  `roleName`      VARCHAR(32) NOT NULL COMMENT '角色名，建号时取 username',
  `itemSeq`       BIGINT      NOT NULL DEFAULT 0 COMMENT '已分配的最大道具 instanceId，只增不减',
  `createTime`    BIGINT      NOT NULL COMMENT '创建时间，毫秒时间戳',
  `lastLoginTime` BIGINT      NOT NULL COMMENT '最近登录时间，毫秒时间戳',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='用户';

-- 已有库手动执行：
-- ALTER TABLE `user` CHANGE `uid` `id` BIGINT NOT NULL AUTO_INCREMENT;
-- ALTER TABLE `user` ADD COLUMN `itemSeq` BIGINT NOT NULL DEFAULT 0 COMMENT '已分配的最大道具 instanceId，只增不减' AFTER `roleName`;
