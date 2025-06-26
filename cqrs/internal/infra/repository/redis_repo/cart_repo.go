package redis_repo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/redis/go-redis/v9"
)

type CartRepoError error

var ErrInsufficientQuantity CartRepoError = errors.New("insufficient quantity")

type CartRepo struct {
	CartCache *redis.Client
}

func NewCartRepo(cartCache *redis.Client) *CartRepo {
	return &CartRepo{CartCache: cartCache}
}

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

// Delta 更新購物車中的 OrderItems（支援 delta 增減）
func (r *CartRepo) Delta(ctx context.Context, userID int, productID string, deltaQuantity int) error {
	itemsKey := generateCartItemKey(userID)

	// 使用 Lua 腳本執行原子增減
	luaScript := `
		local key = KEYS[1]
		local product_id = ARGV[1]
		local delta = tonumber(ARGV[2])
		
		-- 如果是扣減操作，先檢查數量是否足夠
		if delta < 0 then
			local current = tonumber(redis.call('HGET', key, product_id) or "0")
			if current + delta < 0 then
				return -2  -- 商品數量不足
			end
			-- 如果扣減後剛好為 0，直接刪除
			if current == -delta then
				redis.call('HDEL', key, product_id)
				return 0
			end
		end

		-- 使用 HINCRBY 進行原子增減
		return redis.call('HINCRBY', key, product_id, delta)
	`

	result, err := r.CartCache.Eval(ctx, luaScript, []string{itemsKey}, productID, deltaQuantity).Result()
	if err == redis.Nil {
		return fmt.Errorf("failed to execute cart operation")
	}
	if err != nil {
		return fmt.Errorf("failed to add item to cart: %w", err)
	}

	// 處理返回值
	switch v := result.(type) {
	case int64:
		if v == -2 {
			return fmt.Errorf("%w product %s", ErrInsufficientQuantity, productID)
		}
		return nil
	default:
		return fmt.Errorf("unexpected result type: %T", result)
	}
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

// Clear 清空購物車
func (r *CartRepo) Clear(ctx context.Context, userID int) error {
	itemsKey := generateCartItemKey(userID)

	// 刪除整個購物車商品 Hash
	err := r.CartCache.Del(ctx, itemsKey).Err()
	if err == redis.Nil {
		return fmt.Errorf("cart %d not found", userID)
	}
	if err != nil {
		return fmt.Errorf("failed to clear cart: %w", err)
	}
	return nil
}
