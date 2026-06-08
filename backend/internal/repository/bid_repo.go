package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"auction/internal/model"

	redisPkg "auction/pkg/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrBidTooFast = errors.New("bidding too fast, please wait")

const (
	keyCurrentPrice  = "auction:%d:current_price"
	keyBidRanking    = "auction:%d:bids"
	keyBidCount      = "auction:%d:bid_count"
	keyAuctionRules  = "auction:%d:rules"
	keyAuctionStatus = "auction:%d:status"
	keyBidLock       = "bid_lock:%d:%d"
	keyUserBid       = "auction:%d:ub:%d" // per-user bid index, O(1) lookup

	// Global ZSET tracking all active auction deadlines.
	// Score = unix timestamp of effective end time, member = auction ID.
	keyDeadlineZSET = "auction:deadlines"
)

type BidResult struct {
	Rank            int64 `json:"rank"`
	CeilingDeal     bool  `json:"ceiling_deal"`
	DelayExtend     bool  `json:"delay_extend"`     // true if deadline was extended
	NewEndTimestamp int64 `json:"new_end_timestamp"` // Unix ms, 0 if unchanged
	FinalDelay      bool  `json:"final_delay"`       // true if this is the 5th (last) extension
}

type BidRepo struct {
	*BaseRepo[model.Bid]
	rdb     *redisPkg.Client // write (master)
	rdbRead *redisPkg.Client // read (replica, may equal rdb)
}

func NewBidRepo(db *gorm.DB, rdb, rdbRead *redisPkg.Client) *BidRepo {
	return &BidRepo{BaseRepo: NewBaseRepo[model.Bid](db), rdb: rdb, rdbRead: rdbRead}
}

// ============================================================
// Auction rule caching
// ============================================================

func (r *BidRepo) CacheAuctionRules(ctx context.Context, session *model.AuctionSession) error {
	rulesKey := fmt.Sprintf(keyAuctionRules, session.ID)
	statusKey := fmt.Sprintf(keyAuctionStatus, session.ID)
	priceKey := fmt.Sprintf(keyCurrentPrice, session.ID)

	endTime := session.PlannedEndTime
	endTSms := endTime.UnixMilli()

	// Redis 3.2: set fields individually (multi-field HSET requires Redis >= 4.0)
	r.rdb.HSet(ctx, rulesKey, "bid_increment", session.BidIncrement)
	r.rdb.HSet(ctx, rulesKey, "ceiling_price", session.CeilingPrice)
	r.rdb.HSet(ctx, rulesKey, "start_price", session.StartPrice)
	r.rdb.HSet(ctx, rulesKey, "delay_seconds", session.DelaySeconds)
	r.rdb.HSet(ctx, rulesKey, "delay_count", 0)
	r.rdb.HSet(ctx, rulesKey, "planned_end", endTime.Format(time.RFC3339Nano))
	r.rdb.HSet(ctx, rulesKey, "end_timestamp", endTSms)
	r.rdb.Set(ctx, statusKey, int(session.Status), 0)
	r.rdb.SetNX(ctx, priceKey, session.StartPrice, 0)
	err := r.rdb.ZAdd(ctx, keyDeadlineZSET, redis.Z{Score: float64(endTSms), Member: session.ID}).Err()
	return err
}

func (r *BidRepo) GetAuctionRules(ctx context.Context, auctionID uint64) (map[string]string, error) {
	return r.rdbRead.HGetAll(ctx, fmt.Sprintf(keyAuctionRules, auctionID)).Result()
}

// RecoverRules re-caches auction rules from MySQL when Redis data is lost.
func (r *BidRepo) RecoverRules(ctx context.Context, auctionID uint64) error {
	var session model.AuctionSession
	if err := r.DB.WithContext(ctx).Where("id = ?", auctionID).First(&session).Error; err != nil {
		return err
	}
	if err := r.CacheAuctionRules(ctx, &session); err != nil {
		return err
	}
	// Also rebuild bid ranking ZSET from MySQL
	return r.RecoverBidsFromMySQL(ctx, auctionID)
}

// bidWithUser is used to recover bids from MySQL with user info joined.
type bidWithUser struct {
	ID        uint64    `gorm:"column:id"`
	AuctionID uint64    `gorm:"column:auction_id"`
	UserID    uint64    `gorm:"column:user_id"`
	Amount    string    `gorm:"column:amount"`
	BidTime   time.Time `gorm:"column:bid_time"`
	Nickname  string    `gorm:"column:nickname"`
	Avatar    string    `gorm:"column:avatar"`
}

// RecoverBidsFromMySQL re-populates the Redis bid ranking ZSET from MySQL.
// Restores all valid bids so the live-room ranking leaderboard works after recovery.
func (r *BidRepo) RecoverBidsFromMySQL(ctx context.Context, auctionID uint64) error {
	rankKey := fmt.Sprintf(keyBidRanking, auctionID)
	priceKey := fmt.Sprintf(keyCurrentPrice, auctionID)
	countKey := fmt.Sprintf(keyBidCount, auctionID)

	var rows []bidWithUser
	if err := r.DB.WithContext(ctx).
		Table("bids").
		Select("bids.id, bids.auction_id, bids.user_id, bids.amount, bids.bid_time, users.nickname, users.avatar").
		Joins("LEFT JOIN users ON users.id = bids.user_id").
		Where("bids.auction_id = ? AND bids.is_valid = 1", auctionID).
		Order("bids.amount DESC").Limit(500).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil // no bids yet
	}

	// Build ZADD members — one per user (keep highest, rows is amount DESC)
	seen := make(map[uint64]bool)
	members := make([]redis.Z, 0, len(rows))
	for i := range rows {
		if seen[rows[i].UserID] {
			continue
		}
		seen[rows[i].UserID] = true
		amount, _ := strconv.ParseFloat(rows[i].Amount, 64)
		member := fmt.Sprintf(`{"id":%d,"auction_id":%d,"user_id":%d,"amount":"%s","bid_time":"%s","nickname":"%s","avatar":"%s"}`,
			rows[i].ID, rows[i].AuctionID, rows[i].UserID, rows[i].Amount, rows[i].BidTime.Format(time.RFC3339Nano),
			escapeJSON(rows[i].Nickname), escapeJSON(rows[i].Avatar))
		members = append(members, redis.Z{Score: amount, Member: member})
	}

	// highest = first after ORDER BY amount DESC (and per-user dedup)
	topBid := rows[0]

	pipe := r.rdb.Pipeline()
	pipe.ZAdd(ctx, rankKey, members...)
	pipe.Set(ctx, priceKey, topBid.Amount, 0)
	pipe.Set(ctx, countKey, int64(len(rows)), 0) // total bids, not unique bidders
	// Rebuild per-user O(1) lookup keys
	for _, z := range members {
		var snap struct{ UserID uint64 `json:"user_id"` }
		json.Unmarshal([]byte(z.Member.(string)), &snap)
		pipe.Set(ctx, fmt.Sprintf(keyUserBid, topBid.AuctionID, snap.UserID), z.Member, 0)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *BidRepo) SetAuctionStatus(ctx context.Context, auctionID uint64, status model.AuctionStatus) error {
	return r.rdb.Set(ctx, fmt.Sprintf(keyAuctionStatus, auctionID), int(status), 0).Err()
}

func (r *BidRepo) GetAuctionStatus(ctx context.Context, auctionID uint64) (int, error) {
	val, err := r.rdbRead.Get(ctx, fmt.Sprintf(keyAuctionStatus, auctionID)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// ============================================================
// Deadline tracking
// ============================================================

func (r *BidRepo) RegisterDeadline(ctx context.Context, auctionID uint64, endTime time.Time) error {
	return r.rdb.ZAdd(ctx, keyDeadlineZSET, redis.Z{
		Score:  float64(endTime.UnixMilli()),
		Member: auctionID,
	}).Err()
}

func (r *BidRepo) RemoveDeadline(ctx context.Context, auctionID uint64) error {
	return r.rdb.ZRem(ctx, keyDeadlineZSET, auctionID).Err()
}

// GetExpiredAuctions returns auction IDs whose deadline has passed (score = Unix ms).
func (r *BidRepo) GetExpiredAuctions(ctx context.Context) ([]uint64, error) {
	now := float64(time.Now().UnixMilli())
	members, err := r.rdbRead.ZRangeByScore(ctx, keyDeadlineZSET, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatFloat(now, 'f', 0, 64),
	}).Result()
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(members))
	for i, m := range members {
		if id, err := strconv.ParseUint(m, 10, 64); err == nil {
			ids[i] = id
		}
	}
	return ids, nil
}

// ExtendDeadline pushes the deadline forward. newEndTS is Unix milliseconds.
func (r *BidRepo) ExtendDeadline(ctx context.Context, auctionID uint64, newEndTS int64) error {
	return r.rdb.ZAdd(ctx, keyDeadlineZSET, redis.Z{
		Score:  float64(newEndTS),
		Member: auctionID,
	}).Err()
}

// ============================================================
// PlaceBid — atomic Lua script
// ============================================================

func (r *BidRepo) PlaceBid(ctx context.Context, bid *model.Bid) (*BidResult, error) {
	priceKey := fmt.Sprintf(keyCurrentPrice, bid.AuctionID)
	rankKey := fmt.Sprintf(keyBidRanking, bid.AuctionID)
	countKey := fmt.Sprintf(keyBidCount, bid.AuctionID)
	rulesKey := fmt.Sprintf(keyAuctionRules, bid.AuctionID)
	statusKey := fmt.Sprintf(keyAuctionStatus, bid.AuctionID)

	// Lua script: validation + delay extension + ranking + ceiling deal + per-user index
	script := redis.NewScript(`
		-- KEYS: [1]=current_price, [2]=ranking(ZSET), [3]=bid_count,
		--        [4]=rules(HASH), [5]=status, [6]=deadlines(ZSET global),
		--        [7]=user_bid (per-user index, O(1) lookup)
		-- ARGV: [1]=bid_amount, [2]=bid_json, [3]=now_unix_ms

		-- 1. Check auction is active
		local status = redis.call('HGET', KEYS[4], '__status_override')
		if not status then
			status = redis.call('GET', KEYS[5])
		end
		if not status or tonumber(status) ~= 1 then
			return redis.error_reply('AUCTION_NOT_ACTIVE')
		end

		-- 2. Read rules — if missing, signal Go to re-cache from MySQL
		local bid_increment = redis.call('HGET', KEYS[4], 'bid_increment')
		if not bid_increment then
			return redis.error_reply('RULES_MISSING')
		end
		local ceiling_price = redis.call('HGET', KEYS[4], 'ceiling_price')
		local start_price = redis.call('HGET', KEYS[4], 'start_price')
		local delay_seconds = redis.call('HGET', KEYS[4], 'delay_seconds')
		local end_timestamp = redis.call('HGET', KEYS[4], 'end_timestamp')

		local amount = tonumber(ARGV[1])
		local current = redis.call('GET', KEYS[1])
		local now = tonumber(ARGV[3])

		-- 3. Ceiling price check (before min bid validation so ceiling deal always works)
		local ceiling_deal = 0
		local ceiling = tonumber(ceiling_price)
		if ceiling > 0 and amount >= ceiling then
			amount = ceiling
			ARGV[1] = tostring(ceiling)
			ceiling_deal = 1
		end

		-- 4. Validate minimum bid (skip if ceiling deal — the bid is already at ceiling)
		if ceiling_deal == 0 then
			if current then
				local min_next = tonumber(current) + tonumber(bid_increment)
				if amount < min_next then
					return redis.error_reply('BID_TOO_LOW')
				end
			else
				if amount < tonumber(start_price) then
					return redis.error_reply('BID_BELOW_START')
				end
			end
		end

		-- 5. Update current price, ranking, and per-user index
		redis.call('SET', KEYS[1], ARGV[1])
		-- Remove user's previous bid so one user = one rank entry
		local old_member = redis.call('GET', KEYS[7])
		if old_member then
			redis.call('ZREM', KEYS[2], old_member)
		end
		redis.call('ZADD', KEYS[2], amount, ARGV[2])
		redis.call('INCR', KEYS[3])
		redis.call('SET', KEYS[7], ARGV[2])

		-- 6. Delay extension: extend if bid within last 10s (fixed window),
		--    add delay_seconds (10-30, configured by seller). Max 5 extensions.
		local delay_extend = 0
		local new_end_timestamp = 0
		local final_delay = 0
		if ceiling_deal == 0 then
			local end_ts = tonumber(end_timestamp)
			local delay_ms = tonumber(delay_seconds) * 1000
			-- Fixed 10s window before end
			local window_start = end_ts - 10000
			if now >= window_start and now < end_ts then
				local delay_count = 0
				local dc = redis.call('HGET', KEYS[4], 'delay_count')
				if type(dc) == 'string' then
					delay_count = tonumber(dc)
				end
				if delay_count < 1 then
					delay_count = delay_count + 1
					local new_end = end_ts + delay_ms
					new_end_timestamp = new_end
					redis.call('HSET', KEYS[4], 'end_timestamp', new_end)
					redis.call('HSET', KEYS[4], 'delay_count', delay_count)
					redis.call('ZADD', KEYS[6], new_end, KEYS[2]:match('auction:(%d+):bids'))
					delay_extend = 1
				end
			end
		end

		-- 7. If ceiling deal, set deadline to now so watcher picks it up immediately
		if ceiling_deal == 1 then
			new_end_timestamp = now
			redis.call('SET', KEYS[5], 2) -- sold
			redis.call('ZADD', KEYS[6], ARGV[3], KEYS[2]:match('auction:(%d+):bids'))
		end

		local rank = redis.call('ZREVRANK', KEYS[2], ARGV[2]) + 1
		return {rank, ceiling_deal, delay_extend, new_end_timestamp, final_delay}
	`)

	bidJSON := fmt.Sprintf(`{"id":0,"auction_id":%d,"user_id":%d,"amount":"%s","bid_time":"%s","nickname":"%s","avatar":"%s"}`,
		bid.AuctionID, bid.UserID, bid.Amount, time.Now().Format(time.RFC3339Nano),
		escapeJSON(bid.Nickname), escapeJSON(bid.Avatar))

	userBidKey := fmt.Sprintf(keyUserBid, bid.AuctionID, bid.UserID)
	keys := []string{priceKey, rankKey, countKey, rulesKey, statusKey, keyDeadlineZSET, userBidKey}

	result, err := script.Run(ctx, r.rdb, keys,
		bid.Amount, bidJSON, time.Now().UnixMilli(),
	).Slice()
	if err != nil && err.Error() == "ERR RULES_MISSING" {
		// Redis lost the rules — recover from MySQL and retry once
		if recoverErr := r.RecoverRules(ctx, bid.AuctionID); recoverErr == nil {
			result, err = script.Run(ctx, r.rdb, keys,
				bid.Amount, bidJSON, time.Now().UnixMilli(),
			).Slice()
		}
	}
	if err != nil {
		switch err.Error() {
		case "AUCTION_NOT_ACTIVE":
			return nil, fmt.Errorf("auction is not active")
		case "BID_TOO_LOW":
			return nil, fmt.Errorf("bid amount too low: must be current price + increment")
		case "BID_BELOW_START":
			return nil, fmt.Errorf("bid amount below starting price")
		default:
			return nil, err
		}
	}

	if len(result) < 5 {
		return nil, fmt.Errorf("unexpected redis result")
	}

	rank, _ := result[0].(int64)
	ceilingDeal, _ := result[1].(int64)
	delayExtend, _ := result[2].(int64)
	newEndTS, _ := result[3].(int64)
	finalDelay, _ := result[4].(int64)

	return &BidResult{
		Rank: rank, CeilingDeal: ceilingDeal == 1,
		DelayExtend: delayExtend == 1, NewEndTimestamp: newEndTS,
		FinalDelay: finalDelay == 1,
	}, nil
}

// ============================================================
// Redis read helpers
// ============================================================

func (r *BidRepo) GetCurrentPrice(ctx context.Context, auctionID uint64) (string, error) {
	val, err := r.rdbRead.Get(ctx, fmt.Sprintf(keyCurrentPrice, auctionID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// UserBidRank holds a user's bid rank info from Redis.
type UserBidRank struct {
	Amount   string
	Rank     int64
	Nickname string
	Avatar   string
}

// GetUserBidInAuction finds a user's bid amount and rank in the ZSET.
// Uses O(1) per-user index when available, falls back to full scan for legacy bids.
// Returns nil if the user hasn't bid.
func (r *BidRepo) GetUserBidInAuction(ctx context.Context, auctionID, userID uint64) (*UserBidRank, error) {
	rankKey := fmt.Sprintf(keyBidRanking, auctionID)
	userKey := fmt.Sprintf(keyUserBid, auctionID, userID)

	// Fast path: O(1) per-user index lookup
	member, err := r.rdbRead.Get(ctx, userKey).Result()
	if err == nil && member != "" {
		score, scoreErr := r.rdbRead.ZScore(ctx, rankKey, member).Result()
		if scoreErr == nil {
			rank, _ := r.rdbRead.ZRevRank(ctx, rankKey, member).Result()
			var snap struct {
				Nickname string `json:"nickname"`
				Avatar   string `json:"avatar"`
			}
			json.Unmarshal([]byte(member), &snap)
			return &UserBidRank{
				Amount:   fmt.Sprintf("%.2f", score),
				Rank:     rank + 1,
				Nickname: snap.Nickname,
				Avatar:   snap.Avatar,
			}, nil
		}
		// Stale index: member not in ZSET → fall through to slow path
	}

	// Slow fallback: full ZSET scan for bids placed before per-user index existed.
	// Will be retired once all active auctions have the index.
	results, err := r.rdbRead.ZRevRangeWithScores(ctx, rankKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	for _, z := range results {
		m, ok := z.Member.(string)
		if !ok {
			continue
		}
		if contains(m, fmt.Sprintf(`"user_id":%d`, userID)) {
			rank, _ := r.rdbRead.ZRevRank(ctx, rankKey, m).Result()
			var snap struct {
				Nickname string `json:"nickname"`
				Avatar   string `json:"avatar"`
			}
			json.Unmarshal([]byte(m), &snap)
			return &UserBidRank{
				Amount:   fmt.Sprintf("%.2f", z.Score),
				Rank:     rank + 1,
				Nickname: snap.Nickname,
				Avatar:   snap.Avatar,
			}, nil
		}
	}
	return nil, nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ListByUser returns all bids placed by a user, ordered by time desc.
func (r *BidRepo) ListByUser(ctx context.Context, userID uint64, page model.PageRequest) ([]model.Bid, int64, error) {
	var bids []model.Bid
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Bid{}).Where("user_id = ?", userID)
	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&bids).Error
	return bids, total, err
}

func (r *BidRepo) GetBidRanking(ctx context.Context, auctionID uint64, topN int64) ([]redis.Z, error) {
	return r.rdbRead.ZRevRangeWithScores(ctx, fmt.Sprintf(keyBidRanking, auctionID), 0, topN-1).Result()
}

func (r *BidRepo) GetBidCount(ctx context.Context, auctionID uint64) (int64, error) {
	val, err := r.rdbRead.Get(ctx, fmt.Sprintf(keyBidCount, auctionID)).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *BidRepo) GetHighestBidder(ctx context.Context, auctionID uint64) (string, error) {
	results, err := r.rdbRead.ZRevRange(ctx, fmt.Sprintf(keyBidRanking, auctionID), 0, 0).Result()
	if err != nil || len(results) == 0 {
		return "", err
	}
	return results[0], nil
}

// ============================================================
// Distributed lock
// ============================================================

func (r *BidRepo) AcquireBidLock(ctx context.Context, auctionID, userID uint64, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, fmt.Sprintf(keyBidLock, auctionID, userID), time.Now().UnixNano(), ttl).Result()
}

func (r *BidRepo) ReleaseBidLock(ctx context.Context, auctionID, userID uint64) error {
	return r.rdb.Del(ctx, fmt.Sprintf(keyBidLock, auctionID, userID)).Err()
}

// ============================================================
// Idempotency (request-level deduplication)
// ============================================================

const keyIdempotent = "idempotent:bid:%d:%s"

// ClaimIdempotencyKey atomically claims an idempotency key. Returns true if first use.
func (r *BidRepo) ClaimIdempotencyKey(ctx context.Context, auctionID uint64, key string) (bool, error) {
	return r.rdb.SetNX(ctx, fmt.Sprintf(keyIdempotent, auctionID, key), "processing", 10*time.Second).Result()
}

// SaveIdempotencyResult stores the bid result for a claimed key (TTL 5 min).
func (r *BidRepo) SaveIdempotencyResult(ctx context.Context, auctionID uint64, key, resultJSON string) error {
	return r.rdb.Set(ctx, fmt.Sprintf(keyIdempotent, auctionID, key), resultJSON, 5*time.Minute).Err()
}

// ReleaseIdempotencyKey removes the key so the client can retry (called on bid failure).
func (r *BidRepo) ReleaseIdempotencyKey(ctx context.Context, auctionID uint64, key string) error {
	return r.rdb.Del(ctx, fmt.Sprintf(keyIdempotent, auctionID, key)).Err()
}

// ============================================================
// Cleanup
// ============================================================

func (r *BidRepo) FlushAuctionCache(ctx context.Context, auctionID uint64) error {
	keys := []string{
		fmt.Sprintf(keyCurrentPrice, auctionID),
		fmt.Sprintf(keyBidRanking, auctionID),
		fmt.Sprintf(keyBidCount, auctionID),
		fmt.Sprintf(keyAuctionRules, auctionID),
		fmt.Sprintf(keyAuctionStatus, auctionID),
	}
	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, keys...)
	pipe.ZRem(ctx, keyDeadlineZSET, auctionID)
	_, err := pipe.Exec(ctx)
	return err
}

// ============================================================
// MySQL fallback (history)
// ============================================================

func (r *BidRepo) ListByAuction(ctx context.Context, auctionID uint64, page model.PageRequest) ([]model.Bid, int64, error) {
	var bids []model.Bid
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Bid{}).Where("auction_id = ?", auctionID)
	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("amount DESC").Find(&bids).Error
	return bids, total, err
}

func (r *BidRepo) GetLatestBid(ctx context.Context, auctionID uint64) (*model.Bid, error) {
	var bid model.Bid
	err := r.DB.WithContext(ctx).
		Where("auction_id = ? AND is_valid = 1", auctionID).
		Order("amount DESC").
		First(&bid).Error
	if err != nil {
		return nil, err
	}
	return &bid, nil
}

// SyncSave persists a bid to MySQL synchronously and returns the generated ID.
func (r *BidRepo) SyncSave(ctx context.Context, bid *model.Bid) error {
	bid.CreatedAt = time.Now()
	return r.DB.WithContext(ctx).Create(bid).Error
}

func (r *BidRepo) BatchCreate(ctx context.Context, bids []*model.Bid) error {
	if len(bids) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Create(&bids).Error
}

// escapeJSON escapes a string for safe inclusion in a JSON value.
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
