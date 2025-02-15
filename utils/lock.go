package utils

import (
	"billing3/database"
	"context"
	"fmt"
	"time"
)

func Lock(s string) error {
	ctx := context.Background()
	for {
		resp := database.RedisClient.SetNX(ctx, "lock_"+s, fmt.Sprintf("%d", time.Now().Unix()), 0)
		if resp.Err() != nil {
			return fmt.Errorf("redis lock: %w", resp.Err())
		}
		if resp.Val() {
			return nil
		}

		// retry
		time.Sleep(time.Second * 5)
	}
}

func Unlock(s string) error {
	ctx := context.Background()
	resp := database.RedisClient.Del(ctx, fmt.Sprintf("lock_%s", s))
	if resp.Err() != nil {
		return fmt.Errorf("redis unlock: %w", resp.Err())
	}
	return nil
}
