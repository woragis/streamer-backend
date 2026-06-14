package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const pingTimeout = 2 * time.Second

type Client struct {
	rdb    *goredis.Client
	url    string
	status string // ok | disabled | down
}

func Connect(url string) (*Client, error) {
	if url == "" {
		return &Client{status: "disabled"}, nil
	}

	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	rdb := goredis.NewClient(opts)
	c := &Client{rdb: rdb, url: url, status: "down"}

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return c, nil
	}
	c.status = "ok"
	return c, nil
}

func (c *Client) Raw() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

func (c *Client) Enabled() bool {
	return c != nil && c.rdb != nil
}

func (c *Client) Status() string {
	if c == nil || c.rdb == nil {
		return "disabled"
	}
	return c.status
}

func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		c.status = "down"
		return err
	}
	c.status = "ok"
	return nil
}

func (c *Client) Close() error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Close()
}
