CREATE TABLE IF NOT EXISTS `account` (
  `uid`           BIGINT      NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(32) NOT NULL COMMENT '登录名',
  `roleName`      VARCHAR(32) NOT NULL COMMENT '角色名',
  `createTime`    BIGINT      NOT NULL COMMENT '创建时间',
  `lastLoginTime` BIGINT      NOT NULL COMMENT '最近登录时间',
  PRIMARY KEY (`uid`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB AUTO_INCREMENT=10001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='账号';
