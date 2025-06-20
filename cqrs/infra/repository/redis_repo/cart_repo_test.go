package redis_repo

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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

func (suite *CartRepoTestSuite) SetupSuite() {
	redisClient, err := redis_client.GetRedisClient(testRedisAddr, redis_client.WithPassword(testRedisPassword))
	require.NoError(suite.T(), err)
	suite.cartRepo = NewCartRepo(cache.NewRedisCache(redisClient, testPrefix))
}
func (suite *CartRepoTestSuite) TestCreateCacheCart() {
	cart := model.Cart{
		UserID: 1,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  2,
			},
		},
		Amount: decimal.NewFromFloat(200.0),
	}

	cartID, err := suite.cartRepo.CreateCacheCart(context.Background(), 1, cart)

	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), uuid.Nil, cartID)
}

func (suite *CartRepoTestSuite) TestGetCacheCart() {
	originalCart := model.Cart{
		UserID: 1,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  2,
			},
			{
				ProductID: "2",
				Quantity:  3,
			},
			{
				ProductID: "3",
				Quantity:  4,
			},
		},
		Amount: decimal.NewFromFloat(200.0),
	}

	// 先創建購物車
	cartID, err := suite.cartRepo.CreateCacheCart(context.Background(), 1, originalCart)
	require.NoError(suite.T(), err)

	// 獲取購物車
	retrievedCart, err := suite.cartRepo.GetCacheCart(context.Background(), 1)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedCart)
	require.Equal(suite.T(), cartID, retrievedCart.CartID)
	require.Equal(suite.T(), 1, retrievedCart.UserID)
	require.Len(suite.T(), retrievedCart.OrderItems, 3)
	require.Equal(suite.T(), uint(1), retrievedCart.OrderItems[0].ProductID)
	require.Equal(suite.T(), 2, retrievedCart.OrderItems[0].Quantity)
	require.Equal(suite.T(), uint(2), retrievedCart.OrderItems[1].ProductID)
	require.Equal(suite.T(), 3, retrievedCart.OrderItems[1].Quantity)
	require.Equal(suite.T(), uint(3), retrievedCart.OrderItems[2].ProductID)
	require.Equal(suite.T(), 4, retrievedCart.OrderItems[2].Quantity)
	require.True(suite.T(), decimal.NewFromFloat(200.0).Equal(retrievedCart.Amount))
}

func (suite *CartRepoTestSuite) TestGetCacheCart_NotFound() {

	// 嘗試獲取不存在的購物車
	cart, err := suite.cartRepo.GetCacheCart(context.Background(), 1)

	require.Error(suite.T(), err)
	require.Nil(suite.T(), cart)
}
