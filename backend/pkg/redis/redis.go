package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Nil = redis.Nil

type Client struct {
	*redis.Client
}

func NewClient(addr, password string, db, poolSize int) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Client{rdb}, nil
}
