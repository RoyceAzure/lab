package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/redis/go-redis/v9" // for redis.Nil
	"github.com/stretchr/testify/require"
)

const (
	testRedisAddr     = "localhost:6379"
	testRedisPassword = "password"
	testPrefix        = "test_prefix"
)

// Helper function to create a new cache instance for each test
func newTestCache(t *testing.T) cache.Cache {
	t.Helper()
	redisClient, err := redis_client.GetRedisClient(testRedisAddr, redis_client.WithPassword(testRedisPassword))
	require.NoError(t, err, "Failed to connect to Redis for testing")
	require.NotNil(t, redisClient, "Redis client should not be nil")

	// Clear all keys with the test prefix before each test function (or specific group) if needed,
	// though t.Cleanup per test function is more granular.
	// For a truly clean slate for each Test function, you might flush or clean keys by pattern here.

	return cache.NewRedisCache(redisClient, testPrefix)
}

func TestRedisCache_Ping(t *testing.T) {
	rCache := newTestCache(t)
	ctx := context.Background()

	res, err := rCache.Ping(ctx)
	require.NoError(t, err)
	require.Equal(t, "PONG", res)
}

func TestRedisCache_Set_And_Get(t *testing.T) {
	rCache := newTestCache(t)
	ctx := context.Background()
	baseKey := "testSetGetKey"
	testKey := fmt.Sprintf("%s:%s", t.Name(), baseKey) // Unique key for this test
	testValue := "hello world"
	ttl := 10 * time.Minute

	t.Cleanup(func() {
		err := rCache.Delete(ctx, testKey)
		require.NoError(t, err, "Cleanup: failed to delete test key")
	})

	// Test Set
	err := rCache.Set(ctx, testKey, testValue, ttl)
	require.NoError(t, err, "Failed to set value")

	// Test Get
	retrievedValue, err := rCache.Get(ctx, testKey)
	require.NoError(t, err, "Failed to get value")
	require.Equal(t, testValue, retrievedValue, "Retrieved value does not match set value")

	// Test Get for a non-existent key
	nonExistentKey := fmt.Sprintf("%s:nonexistent", t.Name())
	_, err = rCache.Get(ctx, nonExistentKey)
	require.Error(t, err, "Expected an error when getting a non-existent key")
	require.Equal(t, redis.Nil, err, "Expected redis.Nil error for non-existent key")
}

func TestRedisCache_Exists(t *testing.T) {
	rCache := newTestCache(t)
	ctx := context.Background()
	baseKey := "testExistsKey"
	testKey := fmt.Sprintf("%s:%s", t.Name(), baseKey)
	testValue := "exists_value"
	ttl := 5 * time.Minute

	t.Cleanup(func() {
		_ = rCache.Delete(ctx, testKey)
	})

	// Check for non-existent key first
	exists, err := rCache.Exists(ctx, testKey)
	require.NoError(t, err, "Error checking existence of non-existent key")
	require.False(t, exists, "Expected key to not exist initially")

	// Set the key
	err = rCache.Set(ctx, testKey, testValue, ttl)
	require.NoError(t, err, "Failed to set value for existence check")

	// Check for existing key
	exists, err = rCache.Exists(ctx, testKey)
	require.NoError(t, err, "Error checking existence of existing key")
	require.True(t, exists, "Expected key to exist after setting")

	// Delete the key
	err = rCache.Delete(ctx, testKey)
	require.NoError(t, err, "Failed to delete key for existence check")

	// Check for non-existent key after deletion
	exists, err = rCache.Exists(ctx, testKey)
	require.NoError(t, err, "Error checking existence after deletion")
	require.False(t, exists, "Expected key to not exist after deletion")
}

func TestRedisCache_Delete(t *testing.T) {
	rCache := newTestCache(t)
	ctx := context.Background()
	baseKey := "testDeleteKey"
	testKey := fmt.Sprintf("%s:%s", t.Name(), baseKey)
	testValue := "delete_me"
	ttl := 5 * time.Minute

	// No t.Cleanup needed here as the purpose of the test is deletion itself.
	// However, ensure the key doesn't exist before starting or after.

	// Test deleting a non-existent key (should not error)
	_ = rCache.Delete(ctx, fmt.Sprintf("%s:nonexistentfordelete", t.Name()))

	// Set a key to be deleted
	err := rCache.Set(ctx, testKey, testValue, ttl)
	require.NoError(t, err, "Failed to set value for deletion test")

	// Confirm it exists
	exists, err := rCache.Exists(ctx, testKey)
	require.NoError(t, err)
	require.True(t, exists, "Key should exist before deletion")

	// Delete the key
	err = rCache.Delete(ctx, testKey)
	require.NoError(t, err, "Failed to delete existing key")

	// Confirm it no longer exists
	exists, err = rCache.Exists(ctx, testKey)
	require.NoError(t, err)
	require.False(t, exists, "Key should not exist after deletion")

	// Try to Get the deleted key (should be redis.Nil)
	_, err = rCache.Get(ctx, testKey)
	require.Error(t, err)
	require.Equal(t, redis.Nil, err, "Getting a deleted key should result in redis.Nil error")
}

// You might want to add tests for MGet, MSet, MDelete, Clear, Keys, Pipeline later
// For example:
/*
func TestRedisCache_MSet_And_MGet(t *testing.T) {
	rCache := newTestCache(t)
	ctx := context.Background()
	basePrefix := t.Name()
	itemsToSet := map[string]any{
		fmt.Sprintf("%s:key1", basePrefix): "value1",
		fmt.Sprintf("%s:key2", basePrefix): 123,
		fmt.Sprintf("%s:key3", basePrefix): true,
	}
	keysToGet := []string{
		fmt.Sprintf("%s:key1", basePrefix),
		fmt.Sprintf("%s:key2", basePrefix),
		fmt.Sprintf("%s:key3", basePrefix),
		fmt.Sprintf("%s:nonexistent", basePrefix),
	}

	t.Cleanup(func() {
		var keysToDelete []string
		for k := range itemsToSet {
			keysToDelete = append(keysToDelete, k)
		}
		if len(keysToDelete) > 0 {
			err := rCache.MDelete(ctx, keysToDelete...)
			require.NoError(t, err, "Cleanup: failed to MDelete test keys")
		}
	})

	err := rCache.MSet(ctx, itemsToSet)
	require.NoError(t, err, "Failed to MSet items")

	retrievedItems, err := rCache.MGet(ctx, keysToGet...)
	require.NoError(t, err, "Failed to MGet items")
	require.Len(t, retrievedItems, len(keysToGet), "MGet returned wrong number of items")

	// Check values (order matters based on keysToGet)
	require.Equal(t, "value1", retrievedItems[0])
	// Note: Redis MGET can return numbers as strings, or your cache impl might handle type conversion.
	// Depending on your Get/MGet implementation for 'any', you might need type assertion or string comparison.
	// For simplicity, assuming Get returns string representations for numbers if not directly stored as int64
	if val, ok := retrievedItems[1].(string); ok { // go-redis often returns strings for numbers
		 require.Equal(t, "123", val)
	} else {
		 require.Equal(t, int64(123), retrievedItems[1]) // If your cache converts
	}
	if val, ok := retrievedItems[2].(string); ok { // go-redis often returns strings for booleans "1" or "0"
		require.Equal(t, "true", val) // or "1" depending on redis config / client behavior
	} else {
		require.Equal(t, true, retrievedItems[2])
	}
	require.Nil(t, retrievedItems[3], "Expected nil for non-existent key in MGet")
}
*/
