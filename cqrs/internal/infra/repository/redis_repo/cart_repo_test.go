package redis_repo

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

const (
	testRedisAddr     = "localhost:6379"
	testRedisPassword = "password"
	testPrefix        = "test_prefix"
)

type CartRepoTestSuite struct {
	suite.Suite
	cartRepo *CartRepo
}

func setupTestRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     testRedisAddr,
		Password: testRedisPassword,
		DB:       1, // 用測試DB
	})
}

func (suite *CartRepoTestSuite) SetupTest() {
	rdb := setupTestRedis()
	rdb.FlushDB(context.Background())
	suite.cartRepo = NewCartRepo(rdb)
}

func TestCartRepoTestSuite(t *testing.T) {
	suite.Run(t, new(CartRepoTestSuite))
}

func (suite *CartRepoTestSuite) TestCreateAndGetCart() {
	ctx := context.Background()
	cart := &model.Cart{
		UserID: 1,
		OrderItems: []model.CartItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 3},
		},
	}
	err := suite.cartRepo.Create(ctx, cart)
	assert.NoError(suite.T(), err)

	// 測試 Get
	got, err := suite.cartRepo.Get(ctx, 1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), cart.UserID, got.UserID)
	assert.Len(suite.T(), got.OrderItems, 2)
}

func (suite *CartRepoTestSuite) TestAddAndDeleteItem() {
	ctx := context.Background()
	cart := &model.Cart{
		UserID:     2,
		OrderItems: []model.CartItem{},
	}
	suite.cartRepo.Create(ctx, cart)

	// Add 商品
	err := suite.cartRepo.Delta(ctx, 2, "p3", 5)
	assert.NoError(suite.T(), err)
	got, _ := suite.cartRepo.Get(ctx, 2)
	assert.Len(suite.T(), got.OrderItems, 1)
	assert.Equal(suite.T(), 5, got.OrderItems[0].Quantity)

	// 減少商品數量
	err = suite.cartRepo.Delta(ctx, 2, "p3", -2)
	assert.NoError(suite.T(), err)
	got, _ = suite.cartRepo.Get(ctx, 2)
	assert.Equal(suite.T(), 3, got.OrderItems[0].Quantity)

	// 減到0會刪除該商品
	err = suite.cartRepo.Delta(ctx, 2, "p3", -3)
	assert.NoError(suite.T(), err)
	got, _ = suite.cartRepo.Get(ctx, 2)
	assert.Len(suite.T(), got.OrderItems, 0)

	// Add 兩個商品再刪除其中一個
	suite.cartRepo.Delta(ctx, 2, "p4", 1)
	suite.cartRepo.Delta(ctx, 2, "p5", 2)
	err = suite.cartRepo.Delete(ctx, 2, "p4")
	assert.NoError(suite.T(), err)
	got, _ = suite.cartRepo.Get(ctx, 2)
	assert.Len(suite.T(), got.OrderItems, 1)
	assert.Equal(suite.T(), "p5", got.OrderItems[0].ProductID)
}

func (suite *CartRepoTestSuite) TestClearCart() {
	ctx := context.Background()
	cart := &model.Cart{
		UserID: 3,
		OrderItems: []model.CartItem{
			{ProductID: "p1", Quantity: 2},
			{ProductID: "p2", Quantity: 3},
		},
	}
	err := suite.cartRepo.Create(ctx, cart)
	assert.NoError(suite.T(), err)

	// 清空購物車
	err = suite.cartRepo.Clear(ctx, 3)
	assert.NoError(suite.T(), err)

	// 取得購物車，應該沒有商品但 userID 還在
	got, err := suite.cartRepo.Get(ctx, 3)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), cart.UserID, got.UserID)
	assert.Len(suite.T(), got.OrderItems, 0)
}

func (suite *CartRepoTestSuite) TestConcurrentCartOperations() {
	// 創建兩個 context：一個用於操作，一個用於最終檢查
	opCtx, opCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer opCancel()

	cart := &model.Cart{
		UserID: 4,
		OrderItems: []model.CartItem{
			{ProductID: "p1", Quantity: 0},
		},
	}
	err := suite.cartRepo.Create(opCtx, cart)
	assert.NoError(suite.T(), err)

	// 設定併發數量
	const (
		numGoroutines = 1000
		addQuantity   = 5
	)

	// 創建 errgroup，它會繼承 timeout context
	g, ctx := errgroup.WithContext(opCtx)

	// 用於計算 insufficient quantity 錯誤的次數
	insufficientCount := int32(0)

	// 啟動增加商品的 goroutines
	for i := 0; i < numGoroutines; i++ {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := suite.cartRepo.Delta(ctx, 4, "p1", addQuantity)
				if err != nil && !errors.Is(err, ErrInsufficientQuantity) {
					return err
				}
				return nil
			}
		})
	}

	// 啟動減少商品的 goroutines
	for i := 0; i < numGoroutines; i++ {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := suite.cartRepo.Delta(ctx, 4, "p1", -addQuantity)
				if err != nil {
					if errors.Is(err, ErrInsufficientQuantity) {
						atomic.AddInt32(&insufficientCount, 1)
						return nil
					}
					return err
				}
				return nil
			}
		})
	}

	// 等待所有 goroutine 完成
	err = g.Wait()

	// 如果是超時錯誤，輸出更多信息
	if err != nil {
		if err == context.DeadlineExceeded {
			suite.T().Logf("Test timed out: some operations did not complete in time")
		}
		suite.T().Logf("Error occurred: %v", err)
		assert.NoError(suite.T(), err)
		return
	}

	// 使用新的 context 來檢查最終結果
	checkCtx := context.Background()
	finalCart, err := suite.cartRepo.Get(checkCtx, 4)
	assert.NoError(suite.T(), err)

	// 計算期望的最終數量：
	// 增加操作總量 - 成功的減少操作總量
	// = (numGoroutines * addQuantity) - ((numGoroutines - insufficientCount) * addQuantity)
	expectedQuantity := numGoroutines*addQuantity - (numGoroutines-int(insufficientCount))*addQuantity
	suite.T().Logf("Insufficient quantity errors: %d", insufficientCount)
	suite.T().Logf("Expected final quantity: %d", expectedQuantity)

	if len(finalCart.OrderItems) > 0 {
		assert.Equal(suite.T(), expectedQuantity, finalCart.OrderItems[0].Quantity)
	} else {
		assert.Equal(suite.T(), 0, expectedQuantity, "Cart should be empty only if expected quantity is 0")
	}
}
