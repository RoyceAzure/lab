package redis_repo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/redis/go-redis/v9"
)

type CartRepo struct {
	CartCache *redis.Client
}

func NewCartRepo(cartCache *redis.Client) *CartRepo {
	return &CartRepo{CartCache: cartCache}
}

// 創建購物車
// 購物車要使用MHSET
// 購物車更新 orderItems 要能支援delta增減
// 若orderItems 數量為0，則刪除該商品
// Create 創建新的購物車並儲存到 Redis

func generateCartItemKey(userID int) string {
	return fmt.Sprintf("cart:%d:items", userID)
}

func generateCartMetaKey(userID int) string {
	return fmt.Sprintf("cart:%d:meta", userID)
}

func (r *CartRepo) Create(ctx context.Context, cart *model.Cart) error {
	metaKey := generateCartMetaKey(cart.UserID)
	itemsKey := generateCartItemKey(cart.UserID)

	// 使用 Lua 腳本確保原子性
	luaScript := `
		redis.call('HSET', ARGV[1], 'user_id', ARGV[2])
		for i = 4, #ARGV, 2 do
			redis.call('HSET', ARGV[3], ARGV[i], ARGV[i+1])
		end
		return 1
	`
	args := []interface{}{
		metaKey,     // ARGV[1]: meta key
		cart.UserID, // ARGV[2]: user_id
		itemsKey,    // ARGV[3]: items key
	}
	for _, item := range cart.OrderItems {
		args = append(args, item.ProductID, item.Quantity)
	}

	_, err := r.CartCache.Eval(ctx, luaScript, []string{}, args...).Result()
	if err != nil {
		return fmt.Errorf("failed to create cart: %w", err)
	}
	return nil
}

// Get 區域性取購物車資訊
func (r *CartRepo) Get(ctx context.Context, userID int) (*model.Cart, error) {
	metaKey := generateCartMetaKey(userID)
	itemsKey := generateCartItemKey(userID)

	// 獲取元資料
	userID, err := r.CartCache.HGet(ctx, metaKey, "user_id").Int()
	if err == redis.Nil {
		return nil, fmt.Errorf("cart %d not found", userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cart meta: %s", err)
	}

	// 獲取商品列表
	items, err := r.CartCache.HGetAll(ctx, itemsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	// 構建 Cart
	cart := &model.Cart{
		UserID: userID,
	}
	for productID, quantityStr := range items {
		quantity, err := strconv.Atoi(quantityStr)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity for product %s: %w", productID, err)
		}
		if quantity > 0 {
			cart.OrderItems = append(cart.OrderItems, model.CartItem{
				ProductID: productID,
				Quantity:  quantity,
			})
		}
	}

	return cart, nil
}

// Add 更新購物車中的 OrderItems（支援 delta 增減）
func (r *CartRepo) Add(ctx context.Context, userID int, productID string, deltaQuantity int) error {
	itemsKey := generateCartItemKey(userID)

	// 使用 Lua 腳本執行原子增減
	luaScript := `
		local key = KEYS[1]
		local product_id = ARGV[1]
		local delta = tonumber(ARGV[2])
		local current = tonumber(redis.call('HGET', key, product_id) or 0)
		local new_quantity = current + delta
		if new_quantity <= 0 then
			redis.call('HDEL', key, product_id)
		else
			redis.call('HSET', key, product_id, new_quantity)
		end
		return new_quantity
	`
	_, err := r.CartCache.Eval(ctx, luaScript, []string{itemsKey}, productID, deltaQuantity).Result()
	if err == redis.Nil {
		return fmt.Errorf("cart %d not found", userID)
	}
	if err != nil {
		return fmt.Errorf("failed to add item to cart: %w", err)
	}
	return nil
}

// Delete 從購物車中刪除指定商品
func (r *CartRepo) Delete(ctx context.Context, userID int, productID string) error {
	itemsKey := generateCartItemKey(userID)

	// 原子刪除商品
	err := r.CartCache.HDel(ctx, itemsKey, productID).Err()
	if err == redis.Nil {
		return fmt.Errorf("cart %d or product %s not found", userID, productID)
	}
	if err != nil {
		return fmt.Errorf("failed to delete item from cart: %w", err)
	}
	return nil
}
