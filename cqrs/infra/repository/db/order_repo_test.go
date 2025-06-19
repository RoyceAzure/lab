package db

import (
	"context"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/cqrs/config"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	redis_cache "github.com/RoyceAzure/lab/rj_redis/pkg/cache/redis"
	"github.com/RoyceAzure/lab/rj_redis/pkg/redis_client"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type OrderRepoTestSuite struct {
	suite.Suite
	db        *gorm.DB
	orderRepo *OrderRepo
	userRepo  *UserRepo
}

const (
	testRedisAddr     = "localhost:6379"
	testRedisPassword = "password"
	testPrefix        = "test_prefix"
)

// SetupSuite 在測試套件開始前執行
func (suite *OrderRepoTestSuite) SetupSuite() {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(suite.T(), err)
	dbDao := NewDbDao(db)

	redisClient, err := redis_client.GetRedisClient(testRedisAddr, redis_client.WithPassword(testRedisPassword))
	require.NoError(suite.T(), err)
	redisCache := redis_cache.NewRedisCache(redisClient, testPrefix)

	orderRepo := NewOrderRepo(dbDao, redisCache)
	userRepo := NewUserRepo(dbDao)
	suite.db = db
	suite.orderRepo = orderRepo
	suite.userRepo = userRepo
}

// SetupTest 在每個測試前執行
func (suite *OrderRepoTestSuite) SetupTest() {
	// 清空資料表
	suite.db.Exec("DELETE FROM order_items")
	suite.db.Exec("DELETE FROM orders")
	suite.db.Exec("DELETE FROM users")
}

// TearDownSuite 在測試套件結束後執行
func (suite *OrderRepoTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

// 創建測試用的用戶
func (suite *OrderRepoTestSuite) createTestUser() *model.User {
	user := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Test St",
	}
	suite.userRepo.CreateUser(user)
	return user
}

func (suite *OrderRepoTestSuite) TestCreateOrder() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}

	err := suite.orderRepo.CreateOrder(order)

	require.NoError(suite.T(), err)
	require.NotZero(suite.T(), order.OrderID)
	require.False(suite.T(), order.CreatedAt.IsZero())
}

func (suite *OrderRepoTestSuite) TestGetOrderByID() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}
	suite.orderRepo.CreateOrder(order)

	foundOrder, err := suite.orderRepo.GetOrderByID(order.OrderID)

	require.NoError(suite.T(), err)
	require.True(suite.T(), order.Amount.Equal(foundOrder.Amount))
	require.Equal(suite.T(), order.UserID, foundOrder.UserID)
}

func (suite *OrderRepoTestSuite) TestGetOrderByID_NotFound() {
	foundOrder, err := suite.orderRepo.GetOrderByID("999")

	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundOrder)
}

func (suite *OrderRepoTestSuite) TestGetOrdersByUserID() {
	user := suite.createTestUser()

	// 創建多個訂單
	orders := []*model.Order{
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(100.0),
			OrderDate: time.Now(),
		},
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(200.0),
			OrderDate: time.Now(),
		},
	}

	for _, order := range orders {
		suite.orderRepo.CreateOrder(order)
	}

	foundOrders, err := suite.orderRepo.GetOrdersByUserID(user.UserID)

	require.NoError(suite.T(), err)
	require.Len(suite.T(), foundOrders, 2)
}

func (suite *OrderRepoTestSuite) TestGetAllOrders() {
	user := suite.createTestUser()

	// 創建多個訂單
	orders := []*model.Order{
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(100.0),
			OrderDate: time.Now(),
		},
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(200.0),
			OrderDate: time.Now(),
		},
	}

	for _, order := range orders {
		suite.orderRepo.CreateOrder(order)
	}

	allOrders, err := suite.orderRepo.GetAllOrders()

	require.NoError(suite.T(), err)
	require.Len(suite.T(), allOrders, 2)
}

func (suite *OrderRepoTestSuite) TestUpdateOrder() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}
	suite.orderRepo.CreateOrder(order)

	// 更新訂單
	order.Amount = decimal.NewFromFloat(150.0)
	err := suite.orderRepo.UpdateOrder(order)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedOrder, _ := suite.orderRepo.GetOrderByID(order.OrderID)
	require.True(suite.T(), decimal.NewFromFloat(150.0).Equal(updatedOrder.Amount))
}

func (suite *OrderRepoTestSuite) TestUpdateOrderFields() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}
	suite.orderRepo.CreateOrder(order)

	updates := map[string]interface{}{
		"amount": decimal.NewFromFloat(200.0),
	}

	err := suite.orderRepo.UpdateOrderFields(order.OrderID, updates)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedOrder, _ := suite.orderRepo.GetOrderByID(order.OrderID)
	require.True(suite.T(), decimal.NewFromFloat(200.0).Equal(updatedOrder.Amount))
}

func (suite *OrderRepoTestSuite) TestDeleteOrder() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}
	suite.orderRepo.CreateOrder(order)

	err := suite.orderRepo.DeleteOrder(order.OrderID)
	require.NoError(suite.T(), err)

	// 驗證軟刪除
	foundOrder, err := suite.orderRepo.GetOrderByID(order.OrderID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundOrder)
}

func (suite *OrderRepoTestSuite) TestHardDeleteOrder() {
	user := suite.createTestUser()
	order := &model.Order{
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(100.0),
		OrderDate: time.Now(),
	}
	suite.orderRepo.CreateOrder(order)

	err := suite.orderRepo.HardDeleteOrder(order.OrderID)
	require.NoError(suite.T(), err)

	// 驗證硬刪除
	foundOrder, err := suite.orderRepo.GetOrderByID(order.OrderID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundOrder)
}

func (suite *OrderRepoTestSuite) TestGetOrdersPaginated() {
	user := suite.createTestUser()

	// 創建 25 個訂單
	for i := 1; i <= 25; i++ {
		order := &model.Order{
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(int64(i * 100)),
			OrderDate: time.Now(),
		}
		suite.orderRepo.CreateOrder(order)
	}

	// 測試第一頁，每頁 10 筆
	orders, total, err := suite.orderRepo.GetOrdersPaginated(1, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 10)
	require.Equal(suite.T(), int64(25), total)

	// 測試第三頁，每頁 10 筆
	orders, total, err = suite.orderRepo.GetOrdersPaginated(3, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 5) // 第三頁只有 5 筆
	require.Equal(suite.T(), int64(25), total)
}

func (suite *OrderRepoTestSuite) TestGetOrdersPaginated_EmptyResult() {
	orders, total, err := suite.orderRepo.GetOrdersPaginated(1, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), orders, 0)
	require.Equal(suite.T(), int64(0), total)
}

func (suite *OrderRepoTestSuite) TestGetUserOrderStats() {
	user := suite.createTestUser()

	// 創建多個訂單
	orders := []*model.Order{
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(100.0),
			OrderDate: time.Now(),
		},
		{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(200.0),
			OrderDate: time.Now(),
		},
	}

	for _, order := range orders {
		suite.orderRepo.CreateOrder(order)
	}

	totalAmount, orderCount, err := suite.orderRepo.GetUserOrderStats(user.UserID)
	require.NoError(suite.T(), err)
	require.True(suite.T(), decimal.NewFromFloat(300.0).Equal(decimal.NewFromFloat(totalAmount)))
	require.Equal(suite.T(), 2, orderCount)
}

func (suite *OrderRepoTestSuite) TestCreateCacheCart() {
	user := suite.createTestUser()
	cart := model.Cart{
		UserID: user.UserID,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  2,
			},
		},
		Amount: decimal.NewFromFloat(200.0),
	}

	cartID, err := suite.orderRepo.CreateCacheCart(context.Background(), user.UserID, cart)

	require.NoError(suite.T(), err)
	require.NotEqual(suite.T(), uuid.Nil, cartID)
}

func (suite *OrderRepoTestSuite) TestGetCacheCart() {
	user := suite.createTestUser()
	originalCart := model.Cart{
		UserID: user.UserID,
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
	cartID, err := suite.orderRepo.CreateCacheCart(context.Background(), user.UserID, originalCart)
	require.NoError(suite.T(), err)

	// 獲取購物車
	retrievedCart, err := suite.orderRepo.GetCacheCart(context.Background(), user.UserID)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedCart)
	require.Equal(suite.T(), cartID, retrievedCart.CartID)
	require.Equal(suite.T(), user.UserID, retrievedCart.UserID)
	require.Len(suite.T(), retrievedCart.OrderItems, 3)
	require.Equal(suite.T(), uint(1), retrievedCart.OrderItems[0].ProductID)
	require.Equal(suite.T(), 2, retrievedCart.OrderItems[0].Quantity)
	require.Equal(suite.T(), uint(2), retrievedCart.OrderItems[1].ProductID)
	require.Equal(suite.T(), 3, retrievedCart.OrderItems[1].Quantity)
	require.Equal(suite.T(), uint(3), retrievedCart.OrderItems[2].ProductID)
	require.Equal(suite.T(), 4, retrievedCart.OrderItems[2].Quantity)
	require.True(suite.T(), decimal.NewFromFloat(200.0).Equal(retrievedCart.Amount))
}

func (suite *OrderRepoTestSuite) TestGetCacheCart_NotFound() {
	user := suite.createTestUser()

	// 嘗試獲取不存在的購物車
	cart, err := suite.orderRepo.GetCacheCart(context.Background(), user.UserID)

	require.Error(suite.T(), err)
	require.Nil(suite.T(), cart)
}

func (suite *OrderRepoTestSuite) TestUpdateCacheCart() {
	user := suite.createTestUser()

	// 先創建一個購物車
	originalCart := model.Cart{
		UserID: user.UserID,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  2,
			},
		},
		Amount: decimal.NewFromFloat(200.0),
	}

	cartID, err := suite.orderRepo.CreateCacheCart(context.Background(), user.UserID, originalCart)
	require.NoError(suite.T(), err)

	// 修改購物車內容
	updatedCart := model.Cart{
		CartID: cartID,
		UserID: user.UserID,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  5, // 修改數量
			},
			{
				ProductID: "2",
				Quantity:  3, // 新增商品
			},
		},
		Amount: decimal.NewFromFloat(500.0), // 修改總金額
	}

	// 更新購物車
	resultCart, err := suite.orderRepo.UpdateCacheCart(context.Background(), user.UserID, updatedCart)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resultCart)

	// 驗證更新後的購物車
	require.Equal(suite.T(), cartID, resultCart.CartID)
	require.Equal(suite.T(), user.UserID, resultCart.UserID)
	require.Len(suite.T(), resultCart.OrderItems, 2)
	require.Equal(suite.T(), uint(1), resultCart.OrderItems[0].ProductID)
	require.Equal(suite.T(), 5, resultCart.OrderItems[0].Quantity)
	require.Equal(suite.T(), uint(2), resultCart.OrderItems[1].ProductID)
	require.Equal(suite.T(), 3, resultCart.OrderItems[1].Quantity)
	require.True(suite.T(), decimal.NewFromFloat(500.0).Equal(resultCart.Amount))

	// 從快取中讀取並驗證
	retrievedCart, err := suite.orderRepo.GetCacheCart(context.Background(), user.UserID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), retrievedCart)
	require.Equal(suite.T(), cartID, retrievedCart.CartID)
	require.Equal(suite.T(), user.UserID, retrievedCart.UserID)
	require.Len(suite.T(), retrievedCart.OrderItems, 2)
	require.Equal(suite.T(), uint(1), retrievedCart.OrderItems[0].ProductID)
	require.Equal(suite.T(), 5, retrievedCart.OrderItems[0].Quantity)
	require.Equal(suite.T(), uint(2), retrievedCart.OrderItems[1].ProductID)
	require.Equal(suite.T(), 3, retrievedCart.OrderItems[1].Quantity)
	require.True(suite.T(), decimal.NewFromFloat(500.0).Equal(retrievedCart.Amount))
}

func (suite *OrderRepoTestSuite) TestUpdateCacheCart_NotFound() {
	user := suite.createTestUser()

	// 嘗試更新不存在的購物車
	updatedCart := model.Cart{
		UserID: user.UserID,
		OrderItems: []model.CartItem{
			{
				ProductID: "1",
				Quantity:  5,
			},
		},
		Amount: decimal.NewFromFloat(500.0),
	}

	resultCart, err := suite.orderRepo.UpdateCacheCart(context.Background(), user.UserID, updatedCart)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), resultCart)
}

// 執行測試套件
func TestOrderServiceTestSuite(t *testing.T) {
	suite.Run(t, new(OrderRepoTestSuite))
}

// 單獨的單元測試範例
func TestNewOrderRepo(t *testing.T) {
	// 獲取資料庫配置
	cfg, err := config.GetPGConfig()
	require.NoError(t, err)

	// 連接到資料庫
	db, err := GetDbConn(cfg.DbName, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPas)
	require.NoError(t, err)

	dbDao := NewDbDao(db)
	repo := NewOrderRepo(dbDao, nil)

	require.NotNil(t, repo)
	require.Equal(t, dbDao, repo.db)
}

// 基準測試範例
func BenchmarkCreateOrder(b *testing.B) {
	// 獲取資料庫配置
	cfg, err := config.GetPGConfig()
	require.NoError(b, err)

	// 連接到資料庫
	db, err := GetDbConn(cfg.DbName, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPas)
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewOrderRepo(dbDao, nil)
	userRepo := NewUserRepo(dbDao)

	// 創建測試用戶
	user := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Test St",
	}
	userRepo.CreateUser(user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		order := &model.Order{
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(float64(i * 100)),
			OrderDate: time.Now(),
		}
		repo.CreateOrder(order)
	}
}
