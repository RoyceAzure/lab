package db

import (
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
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

	orderRepo := NewOrderRepo(dbDao)
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
