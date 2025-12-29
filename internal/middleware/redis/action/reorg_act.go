package redisclient

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func PauseIndexer(ctx context.Context, r *redis.Client) error {
	return r.Set(ctx, "indexer:paused", "1", 0).Err()
}

func ResumeIndexer(ctx context.Context, r *redis.Client) error {
	return r.Set(ctx, "indexer:paused", "0", 0).Err()
}

func PublishReorg(ctx context.Context, reorgBlock uint64, r *redis.Client) error {
	return r.XAdd(ctx, &redis.XAddArgs{
		Stream: "reorg_jobs",
		Values: map[string]interface{}{
			"reorg_block": reorgBlock,
			"ts":          time.Now().Unix(),
		},
	}).Err()
}
