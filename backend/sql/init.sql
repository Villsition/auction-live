-- ============================================================
-- 直播竞拍系统 - 数据库初始化脚本
-- Database: auction
-- ============================================================

CREATE DATABASE IF NOT EXISTS `auction`
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE `auction`;

-- ============================================================
-- 1. 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS `users` (
  `id`            BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(32)      NOT NULL DEFAULT ''   COMMENT '用户名',
  `password_hash` VARCHAR(256)     NOT NULL DEFAULT ''   COMMENT 'bcrypt密码哈希',
  `nickname`      VARCHAR(64)      NOT NULL DEFAULT ''   COMMENT '昵称',
  `avatar`        VARCHAR(512)     NOT NULL DEFAULT ''   COMMENT '头像URL',
  `role`          TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=普通用户 1=主播/商家 2=管理员',
  `status`        TINYINT UNSIGNED NOT NULL DEFAULT 1    COMMENT '0=禁用 1=正常',
  `created_at`    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_role` (`role`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';


-- ============================================================
-- 2. 商品分类表
-- ============================================================
CREATE TABLE IF NOT EXISTS `categories` (
  `id`           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `name`         VARCHAR(64)      NOT NULL DEFAULT ''   COMMENT '分类名称',
  `parent_id`    BIGINT UNSIGNED  NOT NULL DEFAULT 0    COMMENT '父分类ID，0=顶级',
  `sort`         INT UNSIGNED     NOT NULL DEFAULT 0    COMMENT '排序',
  `status`       TINYINT UNSIGNED NOT NULL DEFAULT 1    COMMENT '0=禁用 1=正常',
  `created_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品分类表';


-- ============================================================
-- 3. 商品表
-- ============================================================
CREATE TABLE IF NOT EXISTS `products` (
  `id`             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `seller_id`      BIGINT UNSIGNED  NOT NULL              COMMENT '商家/主播ID',
  `category_id`    BIGINT UNSIGNED  NOT NULL DEFAULT 0    COMMENT '分类ID',
  `title`          VARCHAR(256)     NOT NULL DEFAULT ''   COMMENT '商品标题',
  `description`    TEXT                                  COMMENT '商品详情(富文本)',
  `cover_image`    VARCHAR(512)     NOT NULL DEFAULT ''   COMMENT '封面图URL',
  `images`         JSON                                  COMMENT '图片列表 ["url1","url2"]',
  `start_price`    DECIMAL(15,2)    NOT NULL DEFAULT 0.00 COMMENT '起拍价(支持0元起拍)',
  `reserve_price`  DECIMAL(15,2)    NOT NULL DEFAULT 0.00 COMMENT '保留价(不公开，低于此价流拍)',
  `ceiling_price`  DECIMAL(15,2)    NOT NULL DEFAULT 0.00 COMMENT '封顶价(0=不设上限，达到自动成交)',
  `bid_increment`  DECIMAL(15,2)    NOT NULL DEFAULT 1.00 COMMENT '加价幅度(每次出价递增单位)',
  `delay_seconds`  INT UNSIGNED     NOT NULL DEFAULT 30   COMMENT '延时秒数(最后N秒有人出价则自动延长)',
  `status`         TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=草稿 1=已上架 2=竞拍中 3=已成交 4=流拍 5=已取消',
  `created_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_seller` (`seller_id`),
  KEY `idx_category` (`category_id`),
  KEY `idx_status` (`status`),
  KEY `idx_seller_status` (`seller_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品表';


-- ============================================================
-- 4. 直播间表
-- ============================================================
CREATE TABLE IF NOT EXISTS `live_rooms` (
  `id`           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `seller_id`    BIGINT UNSIGNED  NOT NULL              COMMENT '主播ID',
  `title`        VARCHAR(256)     NOT NULL DEFAULT ''   COMMENT '直播间标题',
  `cover_image`  VARCHAR(512)     NOT NULL DEFAULT ''   COMMENT '封面图',
  `stream_url`   VARCHAR(512)     NOT NULL DEFAULT ''   COMMENT '推流地址(第三方直播流)',
  `status`       TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=未开播 1=直播中 2=已结束',
  `online_count` INT UNSIGNED     NOT NULL DEFAULT 0    COMMENT '当前在线人数',
  `started_at`   DATETIME         DEFAULT NULL          COMMENT '开播时间',
  `ended_at`     DATETIME         DEFAULT NULL          COMMENT '下播时间',
  `created_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_seller` (`seller_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间表';


-- ============================================================
-- 5. 竞拍场次表
--   一场直播可以有多个竞拍商品（顺序拍卖）
-- ============================================================
CREATE TABLE IF NOT EXISTS `auction_sessions` (
  `id`             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `room_id`        BIGINT UNSIGNED  NOT NULL              COMMENT '直播间ID',
  `product_id`     BIGINT UNSIGNED  NOT NULL              COMMENT '商品ID',
  `start_price`    DECIMAL(15,2)    NOT NULL              COMMENT '起拍价(冗余)',
  `current_price`  DECIMAL(15,2)    NOT NULL              COMMENT '当前最高价',
  `ceiling_price`  DECIMAL(15,2)    NOT NULL DEFAULT 0.00 COMMENT '封顶价(冗余，0=不设上限)',
  `bid_increment`  DECIMAL(15,2)    NOT NULL              COMMENT '加价幅度(冗余)',
  `delay_seconds`  INT UNSIGNED     NOT NULL DEFAULT 30   COMMENT '延时秒数(10-30可配)',
  `start_time`     DATETIME         DEFAULT NULL          COMMENT '竞拍开始时间',
  `planned_end_time` DATETIME       DEFAULT NULL          COMMENT '计划结束时间',
  `actual_end_time`  DATETIME       DEFAULT NULL          COMMENT '实际结束时间(含延时)',
  `winner_id`      BIGINT UNSIGNED  DEFAULT NULL          COMMENT '成交用户ID',
  `final_price`    DECIMAL(15,2)    DEFAULT NULL          COMMENT '成交价',
  `bid_count`      INT UNSIGNED     NOT NULL DEFAULT 0    COMMENT '出价次数',
  `sort_order`     INT UNSIGNED     NOT NULL DEFAULT 0    COMMENT '场次排序(一场直播多件商品)',
  `cancel_reason`  VARCHAR(512)     NOT NULL DEFAULT ''   COMMENT '取消原因',
  `cancelled_by`   BIGINT UNSIGNED  DEFAULT NULL          COMMENT '取消操作人ID',
  `cancelled_at`   DATETIME         DEFAULT NULL          COMMENT '取消时间',
  `status`         TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=待开始 1=进行中 2=已成交 3=已流拍 4=已取消',
  `created_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_room` (`room_id`),
  KEY `idx_product` (`product_id`),
  KEY `idx_status` (`status`),
  KEY `idx_status_end` (`status`, `planned_end_time`),
  KEY `idx_room_sort` (`room_id`, `sort_order`),
  KEY `idx_winner` (`winner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞拍场次表';


-- ============================================================
-- 6. 出价记录表 (核心表，写入频繁)
-- ============================================================
CREATE TABLE IF NOT EXISTS `bids` (
  `id`           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `auction_id`   BIGINT UNSIGNED  NOT NULL              COMMENT '竞拍场次ID',
  `user_id`      BIGINT UNSIGNED  NOT NULL              COMMENT '出价用户ID',
  `amount`       DECIMAL(15,2)    NOT NULL              COMMENT '出价金额',
  `bid_time`     DATETIME(3)      NOT NULL              COMMENT '出价时间(毫秒精度)',
  `client_ip`    VARCHAR(64)      NOT NULL DEFAULT ''   COMMENT '客户端IP',
  `is_valid`     TINYINT UNSIGNED NOT NULL DEFAULT 1    COMMENT '0=无效(被撤回/超时) 1=有效',
  `created_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_auction_amount` (`auction_id`, `amount` DESC),
  KEY `idx_user` (`user_id`),
  KEY `idx_auction_user` (`auction_id`, `user_id`),
  KEY `idx_auction_time` (`auction_id`, `bid_time` DESC),
  KEY `idx_bid_time` (`bid_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='出价记录表';


-- ============================================================
-- 7. 订单表
-- ============================================================
CREATE TABLE IF NOT EXISTS `orders` (
  `id`             BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `order_no`       VARCHAR(32)      NOT NULL DEFAULT ''   COMMENT '订单编号',
  `auction_id`     BIGINT UNSIGNED  NOT NULL              COMMENT '竞拍场次ID',
  `bid_id`         BIGINT UNSIGNED  NOT NULL              COMMENT '中标出价ID',
  `buyer_id`       BIGINT UNSIGNED  NOT NULL              COMMENT '买家ID',
  `seller_id`      BIGINT UNSIGNED  NOT NULL              COMMENT '卖家ID',
  `product_id`     BIGINT UNSIGNED  NOT NULL              COMMENT '商品ID',
  `amount`         DECIMAL(15,2)    NOT NULL              COMMENT '成交金额',
  `status`         TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=待支付 1=已支付 2=已发货 3=已完成 4=已取消 5=已退款',
  `paid_at`        DATETIME         DEFAULT NULL          COMMENT '支付时间',
  `refunded_at`    DATETIME         DEFAULT NULL          COMMENT '退款时间',
  `created_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_no` (`order_no`),
  UNIQUE KEY `uk_bid_id` (`bid_id`),
  KEY `idx_buyer` (`buyer_id`),
  KEY `idx_seller` (`seller_id`),
  KEY `idx_status` (`status`),
  KEY `idx_created` (`created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单表';


-- ============================================================
-- 8. 支付记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS `payment_records` (
  `id`               BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `order_id`         BIGINT UNSIGNED  NOT NULL              COMMENT '订单ID',
  `user_id`          BIGINT UNSIGNED  NOT NULL              COMMENT '付款用户ID',
  `transaction_no`   VARCHAR(64)      NOT NULL DEFAULT ''   COMMENT '第三方交易流水号',
  `amount`           DECIMAL(15,2)    NOT NULL              COMMENT '支付金额',
  `status`           TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=待支付 1=成功 2=失败 3=已退款',
  `paid_at`          DATETIME         DEFAULT NULL          COMMENT '支付完成时间',
  `created_at`       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_transaction` (`transaction_no`),
  KEY `idx_order` (`order_id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付记录表';


-- ============================================================
-- 9. 消息推送记录表 (系统通知/竞拍提醒)
-- ============================================================
CREATE TABLE IF NOT EXISTS `notifications` (
  `id`           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `user_id`      BIGINT UNSIGNED  NOT NULL              COMMENT '接收用户ID',
  `title`        VARCHAR(256)     NOT NULL DEFAULT ''   COMMENT '通知标题',
  `content`      TEXT                                  COMMENT '通知内容',
  `type`         TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=系统通知 1=竞拍提醒 2=出价被超 3=成交通知',
  `related_id`   BIGINT UNSIGNED  NOT NULL DEFAULT 0    COMMENT '关联业务ID',
  `is_read`      TINYINT UNSIGNED NOT NULL DEFAULT 0    COMMENT '0=未读 1=已读',
  `created_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_read` (`user_id`, `is_read`),
  KEY `idx_created` (`created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息推送记录表';


-- ============================================================
-- 10. 竞拍操作日志表 (审计追溯)
--    记录开始/暂停/取消/封顶成交等关键操作
-- ============================================================
CREATE TABLE IF NOT EXISTS `auction_logs` (
  `id`           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `auction_id`   BIGINT UNSIGNED  NOT NULL              COMMENT '竞拍场次ID',
  `operator_id`  BIGINT UNSIGNED  NOT NULL              COMMENT '操作人ID',
  `action`       VARCHAR(32)      NOT NULL DEFAULT ''   COMMENT '操作类型: start/pause/resume/cancel/ceiling_deal/timeout_deal/reserve_fail',
  `detail`       JSON                                  COMMENT '操作详情(快照/备注)',
  `created_at`   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_auction` (`auction_id`),
  KEY `idx_auction_action` (`auction_id`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞拍操作日志表';

 ALTER TABLE `payment_records` DROP COLUMN `pay_method`;