package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	apiURL := flag.String("api", "http://localhost:8081", "API base URL")
	room := flag.Int("room", 5, "Room ID")
	concurrent := flag.Int("c", 100, "Concurrent connections")
	duration := flag.Int("d", 30, "Duration in seconds")
	flag.Parse()

	// Login to get token
	resp, err := http.Post(*apiURL+"/api/auth/login",
		"application/json",
		strings.NewReader(`{"username":"buyer1","password":"123456"}`))
	if err != nil {
		log.Fatal("login failed:", err)
	}
	defer resp.Body.Close()
	var loginRes struct {
		Data struct{ Token string }
	}
	json.NewDecoder(resp.Body).Decode(&loginRes)

	wsURL := fmt.Sprintf("ws://localhost:8081/api/ws?token=%s&room_id=%d", loginRes.Data.Token, *room)
	log.Printf("Connecting %d clients to room %d for %ds", *concurrent, *room, *duration)

	var connected, received int64
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < *concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				return
			}
			defer conn.Close()
			atomic.AddInt64(&connected, 1)
			conn.SetReadDeadline(time.Now().Add(time.Duration(*duration) * time.Second))
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
				atomic.AddInt64(&received, 1)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start).Seconds()

	fmt.Printf("\n=== WebSocket Benchmark Results ===\n")
	fmt.Printf("Target:       %d concurrent clients\n", *concurrent)
	fmt.Printf("Connected:    %d\n", connected)
	fmt.Printf("Duration:     %.1fs\n", elapsed)
	fmt.Printf("Total msgs:   %d\n", received)
	fmt.Printf("Msg/sec:      %.0f\n", float64(received)/elapsed)
	fmt.Printf("Avg msg/s/client: %.1f\n", float64(received)/elapsed/float64(connected))
}
