package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Transaction 模型
type Transaction struct {
	ID              uint    `json:"id"`
	UserID          uint    `json:"user_id"`
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"transaction_type"`
	CreatedAt       string  `json:"created_at"`
}

// Redis & MySQL 客戶端
var (
	redisClient *redis.Client
	db          *gorm.DB
	ctx         = context.Background()
)

// 初始化 Redis & MySQL
func init() {
	// 初始化 Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 連接 MySQL
	var err error
	dsn := "user:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
}

// GetUserTransactions 獲取交易記錄，使用 Redis 快取
func GetUserTransactions(userID int) ([]Transaction, error) {
	cacheKey := fmt.Sprintf("transaction:user:%d", userID)
	lockKey := fmt.Sprintf("lock:%s", cacheKey)

	// 第一次嘗試從快取獲取
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// 快取命中，解析並返回
		var transactions []Transaction
		if err := json.Unmarshal([]byte(cachedData), &transactions); err != nil {
			log.Printf("Failed to unmarshal cached data: %v", err)
		} else {
			log.Printf("Cache hit for user %d", userID)
			return transactions, nil
		}
	}

	// 快取未命中，嘗試獲取分散式鎖
	lockValue := generateLockValue()
	lockAcquired, err := redisClient.SetNX(ctx, lockKey, lockValue, 10*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %v", err)
	}

	if !lockAcquired {
		// 未獲得鎖，等待並重試獲取快取
		log.Printf("Lock not acquired for user %d, waiting for cache update", userID)
		return waitForCacheUpdate(cacheKey, userID)
	}

	// 獲得鎖，負責查詢資料庫並更新快取
	defer releaseLock(lockKey, lockValue)

	// 雙重檢查：防止鎖等待期間其他請求已更新快取
	cachedData, err = redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var transactions []Transaction
		if err := json.Unmarshal([]byte(cachedData), &transactions); err == nil {
			log.Printf("Cache hit after lock acquisition for user %d", userID)
			return transactions, nil
		}
	}

	// 從資料庫查詢
	log.Printf("Querying database for user %d", userID)
	var transactions []Transaction
	if err := db.Where("user_id = ?", userID).Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("database query failed: %v", err)
	}

	// 序列化並寫入快取
	jsonData, err := json.Marshal(transactions)
	if err != nil {
		log.Printf("Failed to marshal transactions: %v", err)
		return transactions, nil // 返回資料但不快取
	}

	// 設置快取，TTL 30分鐘
	if err := redisClient.Set(ctx, cacheKey, jsonData, 30*time.Minute).Err(); err != nil {
		log.Printf("Failed to cache data: %v", err)
	} else {
		log.Printf("Data cached for user %d", userID)
	}

	return transactions, nil
}

// 生成唯一的鎖值
func generateLockValue() string {
	bigInt := new(big.Int).SetInt64(time.Now().UnixNano())
	i, _ := rand.Int(rand.Reader, bigInt)
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), i.String())
}

// 等待快取更新
func waitForCacheUpdate(cacheKey string, userID int) ([]Transaction, error) {
	maxRetries := 3
	retryDelay := 50 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryDelay)

		cachedData, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var transactions []Transaction
			if err := json.Unmarshal([]byte(cachedData), &transactions); err == nil {
				log.Printf("Cache hit after retry %d for user %d", i+1, userID)
				return transactions, nil
			}
		}

		// 指數退避
		retryDelay *= 2
	}

	// 重試失敗，降級查詢資料庫
	log.Printf("Cache wait timeout for user %d, fallback to database", userID)
	var transactions []Transaction
	if err := db.Where("user_id = ?", userID).Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("fallback database query failed: %v", err)
	}

	return transactions, nil
}

// 安全釋放鎖
func releaseLock(lockKey, lockValue string) {
	script := `
   	if redis.call("GET", KEYS[1]) == ARGV[1] then
   		return redis.call("DEL", KEYS[1])
   	else
   		return 0
   	end
   `

	result, err := redisClient.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	if err != nil {
		log.Printf("Failed to release lock: %v", err)
	} else if result.(int64) == 1 {
		log.Printf("Lock released successfully")
	} else {
		log.Printf("Lock was not owned by this instance")
	}
}

func main() {
	transactions, err := GetUserTransactions(1)
	if err != nil {
		log.Println("Error:", err)
	} else {
		fmt.Printf("Transactions: %+v\n", transactions)
	}
}
