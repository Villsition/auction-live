# Auction — 直播间竞拍系统

基于 WebSocket 实时通信的直播拍卖平台，支持高并发出价、毫秒级倒计时、封顶成交、延时拍卖。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23 + Gin + GORM |
| 前端 | React 18 + TypeScript + Vite 5 |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7.0 |
| 实时通信 | WebSocket（gorilla/websocket） |
| 可观测性 | Prometheus Metrics + Health Check |

## 项目结构

```
auction/
├── backend/
│   ├── cmd/
│   │   ├── server/main.go           # 服务入口
│   │   └── bidbench/main.go         # 高并发压测工具
│   ├── internal/
│   │   ├── config/                  # 配置加载（viper + 环境变量）
│   │   ├── model/                   # 数据模型
│   │   ├── handler/                 # HTTP 处理器
│   │   │   ├── seller_handler.go    # 商家端：商品/竞拍管理
│   │   │   ├── buyer_handler.go     # 用户端：直播间/排名/出价历史
│   │   │   ├── bid_handler.go       # 出价接口
│   │   │   ├── ws_handler.go        # WebSocket 鉴权连接
│   │   │   └── health_handler.go    # 健康检查
│   │   ├── service/                 # 业务逻辑层
│   │   ├── repository/              # 数据访问层（MySQL + Redis）
│   │   ├── middleware/              # JWT 认证 + CORS + 日志
│   │   ├── router/                  # 路由定义
│   │   ├── ws/                      # WebSocket Hub + Client
│   │   ├── scheduler/               # 定时任务（成交检测 / 点赞落库）
│   │   └── metrics/                 # Prometheus 指标
│   ├── pkg/                         # 公共工具包
│   │   ├── redis/                   # Redis 客户端
│   │   ├── response/                # 统一响应格式
│   │   └── upload/                  # 图片/视频上传
│   ├── sql/
│   │   ├── init.sql                 # 初始化建表
│   │   └── migrate.sql             # 增量迁移
│   └── config/
│       └── config.example.yaml      # 配置模板
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── buyer/               # 买家端页面
│   │   │   │   ├── LiveRoom.tsx     # 直播间（核心页面）
│   │   │   │   ├── LiveRoomList.tsx # 直播间列表
│   │   │   │   ├── Orders.tsx       # 我的订单
│   │   │   │   └── BidHistory.tsx   # 竞拍历史
│   │   │   └── seller/              # 卖家端页面
│   │   │       ├── Dashboard.tsx    # 工作台
│   │   │       ├── Products.tsx     # 商品管理
│   │   │       ├── LiveRooms.tsx    # 直播间设置
│   │   │       └── Orders.tsx       # 订单管理
│   │   ├── components/
│   │   │   ├── Countdown.tsx        # 毫秒倒计时
│   │   │   └── RankingList.tsx      # FLIP 动画排行榜
│   │   ├── hooks/
│   │   │   └── useWebSocket.ts      # WebSocket 连接 + 自动重连
│   │   ├── store/
│   │   │   └── AuthContext.tsx       # 认证状态管理
│   │   ├── api/
│   │   │   └── index.ts             # 前端 API 封装
│   │   └── types/
│   │       └── index.ts             # TypeScript 类型定义
│   └── public/video/                # 直播间背景视频
└── README.md
```

## 依赖环境

| 软件 | 最低版本 | 用途 |
|------|----------|------|
| Go | 1.21+ | 后端编译运行 |
| Node.js | 18+ | 前端开发构建 |
| MySQL | 5.7+ | 核心业务数据 |
| Redis | 6.0+ | 高并发缓存与原子出价 |

## 启动步骤

### 1. 初始化数据库

执行 `backend/sql/init.sql` 创建表结构：

```bash
mysql -u root -p auction < backend/sql/init.sql
mysql -u root -p auction < backend/sql/migrate.sql
```

### 2. 配置环境变量

复制配置模板并修改：

```bash
cp backend/config/config.example.yaml backend/config/config.yaml
```

编辑 `config.yaml`，填写 MySQL 和 Redis 连接信息：

```yaml
db:
  host: 127.0.0.1
  port: 3306
  user: root
  password: your-db-password
  database: auction
redis:
  host: 127.0.0.1
  port: 6379
  password: ""
jwt:
  secret: your-jwt-secret
```

也支持通过环境变量覆盖（优先级高于配置文件）：

| 环境变量 | 对应配置 |
|----------|----------|
| `DB_HOST` | MySQL 地址 |
| `DB_PASSWORD` | MySQL 密码 |
| `REDIS_HOST` | Redis 地址 |
| `REDIS_PASSWORD` | Redis 密码 |
| `JWT_SECRET` | JWT 签名密钥 |
| `SERVER_PORT` | 服务端口 |

### 3. 启动后端

```bash
cd backend
go mod tidy
go run cmd/server/main.go
# 或编译后运行
go build -o auction-server cmd/server/main.go
./auction-server
```

### 4. 启动前端

```bash
cd frontend
npm install
npm run dev
```

浏览器访问 `http://localhost:3000` 即可。

### 部署到生产环境

```bash
# 构建前端静态文件
cd frontend
npm run build

# 编译后端
cd backend
go build -o auction-server cmd/server/main.go

# Nginx 配置示例（见部署文档）
# 启动后端（systemd 服务或 nohup）
nohup ./auction-server > server.log 2>&1 &
```

## 配置说明

| 配置项 | 文件 | 说明 |
|--------|------|------|
| 数据库连接 | `config/config.yaml` → `db` | MySQL 地址、用户名、密码 |
| Redis 连接 | `config/config.yaml` → `redis` | Redis 地址、密码、连接池 |
| JWT 密钥 | `config/config.yaml` → `jwt.secret` | 用户认证令牌签名 |
| 延时时间 | 商品属性 `delay_seconds` | 竞拍最后 N 秒出价自动延时（10-30s） |
| 竞拍时长 | 商品属性 `duration_min` | 单次竞拍持续分钟数 |
| 前端端口 | `frontend/vite.config.ts` → `server.port` | 开发服务器端口（默认 3000） |

## 竞拍核心链路

```
商家创建商品 → 创建竞拍场次 → 开始竞拍
    ↓
规则缓存到 Redis Hash
    ↓
用户出价 → Redis Lua 原子校验（加价/封顶/延时）
    ↓
ZSET 更新排名 → WebSocket 广播（出价/排名/超越/延时）
    ↓
AuctionWatcher 每秒扫描到期拍卖 → 成交/流拍 → 生成订单
```

## 高并发压测

使用内置压测工具模拟 100 人同时出价：

```bash
cd backend
go build -o bidbench cmd/bidbench/main.go
./bidbench -api http://localhost:8080 -auction <拍卖ID> -c 100 -n 5
```

参数：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-api` | 后端 API 地址 | `http://localhost:8080` |
| `-auction` | 竞拍场次 ID | 必填 |
| `-c` | 并发用户数 | 50 |
| `-n` | 每用户出价次数 | 5 |

## 可观测性

| 端点 | 用途 |
|------|------|
| `GET /api/health` | 健康检查（MySQL + Redis 连通性） |
| `GET /metrics` | Prometheus 指标（QPS/延迟/WS连接数/出价统计） |

## 高并发保障

| 机制 | 实现 |
|------|------|
| 原子出价 | Redis Lua 脚本单线程串行执行 |
| 分布式锁 | `bid_lock` SetNX 1s TTL |
| 幂等去重 | `idempotency_key` 10s TTL |
| 数据一致性 | MySQL 事务（拍卖+商品+订单同步） |
| Redis 恢复 | `RULES_MISSING` → 自动从 MySQL 重建 |
| 断连重连 | 前端指数退避 1s→30s |
| 房间隔离 | WebSocket 房间级广播（多直播间互不干扰） |
| Panic 恢复 | watcher goroutine 自动重启 |
