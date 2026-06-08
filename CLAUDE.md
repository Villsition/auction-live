# Auction — 直播间竞拍系统

## 技术栈
Go + Gin + GORM + MySQL + Redis + WebSocket (gorilla/websocket)

## 项目结构
```
backend/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── config/                 # 配置加载
│   ├── model/                  # 数据模型
│   ├── repository/             # 数据层（MySQL + Redis）
│   ├── service/                # 业务逻辑
│   ├── handler/                # HTTP 处理器
│   │   ├── seller_handler.go   # 商家端（商品/竞拍管理）
│   │   ├── buyer_handler.go    # 用户端（直播间/排名/出价历史）
│   │   ├── bid_handler.go      # 出价
│   │   └── ws_handler.go       # WebSocket 连接
│   ├── middleware/             # JWT 认证 + 角色校验
│   ├── router/                 # 路由定义
│   ├── scheduler/              # 定时任务（竞拍成交检测）
│   └── ws/                     # WebSocket Hub + Client
└── pkg/                        # 工具包（Redis/响应/文件上传）
```

## 竞拍核心链路
1. 商家创建商品 → 创建竞拍场次（配置规则） → 开始竞拍
2. 开始竞拍时，规则缓存到 Redis Hash
3. 用户出价 → Redis Lua 脚本原子校验（加价幅度/封顶价/延时）→ ZSET 排名 → 异步写 MySQL
4. WebSocket 广播出价事件、排行榜变更、被超越通知
5. AuctionWatcher 每秒扫描到期竞拍 → 成交/流拍 → 生成订单

## Redis 数据结构
- `auction:{id}:current_price` — String，当前最高价
- `auction:{id}:bids` — ZSET，score=金额，member=JSON（id/user_id/nickname/avatar/amount/bid_time）
- `auction:{id}:bid_count` — String，出价次数
- `auction:{id}:rules` — Hash，竞拍规则
- `auction:{id}:status` — String，竞拍状态
- `auction:deadlines` — ZSET，全局到期时间索引
- `bid_lock:{auction_id}:{user_id}` — String，用户出价锁（2s TTL）

## 已知待修复项
参考 memory/backend-audit-and-fixes.md
