package redis_repo

import (
	"context"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
	err := suite.cartRepo.Add(ctx, 2, "p3", 5)
	assert.NoError(suite.T(), err)
	got, _ := suite.cartRepo.Get(ctx, 2)
	assert.Len(suite.T(), got.OrderItems, 1)
	assert.Equal(suite.T(), 5, got.OrderItems[0].Quantity)

	// 減少商品數量
	err = suite.cartRepo.Add(ctx, 2, "p3", -2)
	assert.NoError(suite.T(), err)
	got, _ = suite.cartRepo.Get(ctx, 2)
	assert.Equal(suite.T(), 3, got.OrderItems[0].Quantity)

	// 減到0會刪除該商品
	err = suite.cartRepo.Add(ctx, 2, "p3", -3)
	assert.NoError(suite.T(), err)
	got, _ = suite.cartRepo.Get(ctx, 2)
	assert.Len(suite.T(), got.OrderItems, 0)

	// Add 兩個商品再刪除其中一個
	suite.cartRepo.Add(ctx, 2, "p4", 1)
	suite.cartRepo.Add(ctx, 2, "p5", 2)
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
