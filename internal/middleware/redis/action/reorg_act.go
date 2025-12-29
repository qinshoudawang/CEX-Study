package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func IsIndexerPaused(ctx context.Context, r *redis.Client) (bool, error) {
	val, err := r.Get(ctx, "indexer:paused").Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // key does not exist, so not paused
		}
		return false, err
	}
	return val == "1", nil
}

func PauseIndexer(ctx context.Context, r *redis.Client) error {
	return r.Set(ctx, "indexer:paused", "1", 0).Err()
}

func ResumeIndexer(ctx context.Context, r *redis.Client) error {
	return r.Set(ctx, "indexer:paused", "0", 0).Err()
}

func PublishReorg(ctx context.Context, blockNumber uint64, r *redis.Client) error {
	return r.XAdd(ctx, &redis.XAddArgs{
		Stream: "reorg_jobs",
		Values: map[string]any{
			"block_number": blockNumber,
			"ts":           time.Now().Unix(),
		},
	}).Err()
}

func FetchReorg(ctx context.Context, r *redis.Client) (string, uint64, error) {
	result, err := r.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"reorg_jobs", "0"},
		Count:   1,
		Block:   -1,
	}).Result()
	if err != nil {
		return "", 0, err
	}
	if len(result) == 0 || len(result[0].Messages) == 0 {
		return "", 0, nil // no new messages
	}

	msg := result[0].Messages[0]
	blockNumberStr, ok := msg.Values["block_number"].(string)
	if !ok {
		return "", 0, err // invalid message format
	}

	var blockNumber uint64
	_, err = fmt.Sscanf(blockNumberStr, "%d", &blockNumber)
	if err != nil {
		return "", 0, err
	}

	return msg.ID, blockNumber, nil
}

func AcknowledgeReorg(ctx context.Context, r *redis.Client, id string) error {
	return r.XDel(ctx, "reorg_jobs", id).Err()
}
