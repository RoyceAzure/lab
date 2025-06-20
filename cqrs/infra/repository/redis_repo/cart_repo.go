package redis_repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/google/uuid"
)

type CartRepo struct {
	CartCache redis_cache.Cache
}

func NewCartRepo(cartCache redis_cache.Cache) *CartRepo {
	return &CartRepo{CartCache: cartCache}
}

// 創建購物車
// 購物車要使用MHSET
// 購物車更新 orderItems 要能支援delta增減
// 若orderItems 數量為0，則刪除該商品
func (s *CartRepo) CreateCacheCart(ctx context.Context, userID int, cart model.Cart) (uuid.UUID, error) {
	//購物車主資料
	cartId := uuid.New()
	cart.CartID = cartId
	cartJSON, err := json.Marshal(cart)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("序列化購物車失敗: %v", err)
	}

	key := fmt.Sprintf("cart:%d", userID)
	err = s.CartCache.Set(ctx, key, cartJSON, 24*time.Hour)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("保存購物車失敗: %v", err)
	}
	return cartId, nil
}

// 取得購物車
func (s *CartRepo) GetCacheCart(ctx context.Context, userID int) (*model.Cart, error) {
	key := fmt.Sprintf("cart:%d", userID)
	cartJSON, err := s.CartCache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("獲取購物車失敗: %v", err)
	}
	var cart model.Cart
	cartJSONStr, ok := cartJSON.(string)
	if !ok {
		return nil, fmt.Errorf("購物車資料格式錯誤")
	}
	err = json.Unmarshal([]byte(cartJSONStr), &cart)
	if err != nil {
		return nil, fmt.Errorf("反序列化購物車失敗: %v", err)
	}

	return &cart, nil
}

// 修改購物車
func (s *CartRepo) UpdateCacheCart(ctx context.Context, userID int, cart model.Cart) (*model.Cart, error) {
	key := fmt.Sprintf("cart:%d", userID)
	_, err := s.CartCache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("獲取購物車失敗: %v", err)
	}
	cartJSON, err := json.Marshal(cart)
	if err != nil {
		return nil, fmt.Errorf("序列化購物車失敗: %v", err)
	}

	err = s.CartCache.Set(ctx, key, cartJSON, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("保存購物車失敗: %v", err)
	}

	return &cart, nil
}
