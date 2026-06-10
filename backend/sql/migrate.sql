-- ============================================================
-- 数据库变更脚本
-- ============================================================

USE `auction`;

-- 删除支付方式字段
ALTER TABLE `payment_records` DROP COLUMN `pay_method`;

-- 用户表：phone → username + password_hash
ALTER TABLE `users`
  DROP INDEX `uk_phone`,
  DROP COLUMN `phone`,
  ADD COLUMN `username` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '用户名' AFTER `id`,
  ADD COLUMN `password_hash` VARCHAR(256) NOT NULL DEFAULT '' COMMENT 'bcrypt密码哈希' AFTER `username`,
  ADD UNIQUE KEY `uk_username` (`username`);

-- 订单表：收货地址 + 支付过期时间
ALTER TABLE `orders`
  ADD COLUMN `address` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '收货地址' AFTER `amount`,
  ADD COLUMN `expires_at` DATETIME DEFAULT NULL COMMENT '支付截止时间' AFTER `status`;

-- 评论表
CREATE TABLE IF NOT EXISTS `comments` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `room_id`    BIGINT UNSIGNED NOT NULL              COMMENT '直播间ID',
  `user_id`    BIGINT UNSIGNED NOT NULL              COMMENT '用户ID',
  `content`    VARCHAR(500)    NOT NULL DEFAULT ''   COMMENT '评论内容',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_room` (`room_id`),
  KEY `idx_room_time` (`room_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播评论表';

-- 直播间点赞总数
ALTER TABLE `live_rooms`
  ADD COLUMN `total_likes` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点赞总数' AFTER `online_count`;

-- watcher 到期扫描性能优化：复合索引覆盖 status + planned_end_time
ALTER TABLE `auction_sessions`
  ADD INDEX `idx_status_end` (`status`, `planned_end_time`);

-- 视频拉流地址
ALTER TABLE `live_rooms`
  ADD COLUMN `pull_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '拉流地址' AFTER `stream_url`;

-- 背景视频
ALTER TABLE `live_rooms`
  ADD COLUMN `bg_video` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '背景视频' AFTER `pull_url`;
