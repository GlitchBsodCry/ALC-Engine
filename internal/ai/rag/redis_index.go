package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

func InitRedisIndex(ctx context.Context, rdb *redis.Client, filename string, dimension int) error {
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}
	indexName := GenerateIndexName(filename)
	_, err := rdb.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "Unknown index name") {
		return fmt.Errorf("check index: %w", err)
	}
	prefix := GenerateIndexNamePrefix(filename)
	createArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", dimension,
		"DISTANCE_METRIC", "COSINE",
	}
	if err := rdb.Do(ctx, createArgs...).Err(); err != nil {
		return fmt.Errorf("create redis index: %w", err)
	}
	return nil
}

// RedisIndexExists returns whether a RediSearch index exists for the given logical filename.
func RedisIndexExists(ctx context.Context, rdb *redis.Client, filename string) (bool, error) {
	if rdb == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	indexName := GenerateIndexName(filename)
	_, err := rdb.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "Unknown index name") {
		return false, nil
	}
	return false, fmt.Errorf("check index: %w", err)
}

func DeleteRedisIndex(ctx context.Context, rdb *redis.Client, filename string) error {
	if rdb == nil {
		return fmt.Errorf("redis client is nil")
	}
	indexName := GenerateIndexName(filename)
	return rdb.Do(ctx, "FT.DROPINDEX", indexName).Err()
}
