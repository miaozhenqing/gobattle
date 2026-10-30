CREATE TABLE IF NOT EXISTS `user` (
  `id`            BIGINT      NOT NULL COMMENT '主键id',
  `username`      VARCHAR(32) NOT NULL DEFAULT '' COMMENT '登录名',
  `roleName`      VARCHAR(32) NOT NULL DEFAULT '' COMMENT '角色名，建号时取 username',
  `createTime`    BIGINT      NOT NULL DEFAULT 0 COMMENT '创建时间',
  `lastLoginTime` BIGINT      NOT NULL DEFAULT 0 COMMENT '最近登录时间',
  `updateTime`    BIGINT      NOT NULL DEFAULT 0 COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='用户';
