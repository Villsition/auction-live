package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "API base URL")
	auction := flag.Int("auction", 0, "Auction ID (optional, overrides room)")
	room := flag.Int("room", 8, "Room ID to auto-detect auction")
	concurrent := flag.Int("c", 50, "Concurrent bidders")
	bidsEach := flag.Int("n", 5, "Bids per bidder")
	flag.Parse()

	// Auto-detect auction from room if not specified
	auctionID := *auction
	baseAmount := 1000
	inc := 1000

	if auctionID == 0 {
		price, increment := getAuctionInfo(*apiURL, *room)
		baseAmount = price
		inc = increment
		if inc < 100 {
			inc = 100
		}
		// Get auction ID from the room
		auctionID = getAuctionID(*apiURL, *room)
		if auctionID == 0 {
			log.Fatal("未找到活跃竞拍，请先开始竞拍或使用 -auction 指定ID")
		}
	}

	if baseAmount < 1000 {
		baseAmount = 1000
	}

	fmt.Printf("竞拍ID: %d, 当前价: %d, 加价幅度: %d\n", auctionID, baseAmount, inc)

	var success, fail int64
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < *concurrent; i++ {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()

			// Pre-registered users: buy1-buy100 (register once via setup script)
		username := fmt.Sprintf("bu%d", uid+1)
			token := loginOrRegister(*apiURL, username, "123456")
			if token == "" {
				return
			}

			for j := 0; j < *bidsEach; j++ {
				amount := baseAmount + inc*(uid+1+j)
				ok := placeBid(*apiURL, auctionID, amount, token)
				if ok {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&fail, 1)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}(i + 1)
	}

	wg.Wait()
	elapsed := time.Since(start).Seconds()

	fmt.Printf("\n=== 竞拍压测结果 ===\n")
	fmt.Printf("并发人数:    %d\n", *concurrent)
	fmt.Printf("每人出价:    %d\n", *bidsEach)
	fmt.Printf("总请求:      %d\n", *concurrent**bidsEach)
	fmt.Printf("成功:        %d\n", success)
	fmt.Printf("失败:        %d\n", fail)
	fmt.Printf("耗时:        %.1fs\n", elapsed)
	fmt.Printf("QPS:         %.0f\n", float64(*concurrent**bidsEach)/elapsed)
}

// getAuctionInfo returns current price and increment for the active auction in a room.
func getAuctionID(apiURL string, roomID int) int {
	resp, err := http.Get(fmt.Sprintf("%s/api/live-rooms/%d/auction", apiURL, roomID))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var r struct {
		Data struct {
			AuctionSession struct {
				ID uint64 `json:"id"`
			} `json:"auction_session"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&r)
	return int(r.Data.AuctionSession.ID)
}

func getAuctionInfo(apiURL string, roomID int) (int, int) {
	resp, err := http.Get(fmt.Sprintf("%s/api/live-rooms/%d/auction", apiURL, roomID))
	if err != nil {
		return 1000, 100
	}
	defer resp.Body.Close()
	var r struct {
		Data struct {
			CurrentPrice    string `json:"current_price"`
			AuctionSession  struct {
				ID           uint64 `json:"id"`
				BidIncrement string `json:"bid_increment"`
			} `json:"auction_session"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&r)
	var price float64
	fmt.Sscanf(r.Data.CurrentPrice, "%f", &price)
	var inc float64
	fmt.Sscanf(r.Data.AuctionSession.BidIncrement, "%f", &inc)
	return int(price), int(inc)
}

func loginOrRegister(apiURL, username, password string) string {
	// Try login first
	body := fmt.Sprintf(`{"username":"%s","password":"%s","force":true}`, username, password)
	resp, err := http.Post(apiURL+"/api/auth/login", "application/json",
		bytes.NewBufferString(body))
	if err == nil {
		defer resp.Body.Close()
		var r struct {
			Data struct{ Token string }
		}
		if json.NewDecoder(resp.Body).Decode(&r) == nil && r.Data.Token != "" {
			return r.Data.Token
		}
	}

	// Register if login fails
	regBody := fmt.Sprintf(`{"username":"%s","password":"%s","nickname":"竞拍者%d"}`, username, password, time.Now().UnixNano()%10000)
	resp2, err := http.Post(apiURL+"/api/auth/register", "application/json",
		bytes.NewBufferString(regBody))
	if err != nil {
		return ""
	}
	defer resp2.Body.Close()
	var r struct {
		Data struct{ Token string }
	}
	json.NewDecoder(resp2.Body).Decode(&r)
	return r.Data.Token
}

func placeBid(apiURL string, auctionID, amount int, token string) bool {
	body := fmt.Sprintf(`{"auction_id":%d,"amount":"%d","idempotency_key":"bench-%d-%d"}`, auctionID, amount, time.Now().UnixNano(), amount)
	req, _ := http.NewRequest("POST", apiURL+"/api/bids", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	// code=0 means success
	return bytes.Contains(respBody, []byte(`"code":0`))
}
