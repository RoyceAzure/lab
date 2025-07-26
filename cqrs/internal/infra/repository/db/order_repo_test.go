package db

import (
	"context"
	"fmt"
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
	suite.userRepo.CreateUser(context.Background(), user)
	return user
}

// 創建測試用的產品
func (suite *OrderRepoTestSuite) createTestProducts(count int) []*model.Product {
	products := make([]*model.Product, count)
	for i := 0; i < count; i++ {
		products[i] = &model.Product{
			ProductID: fmt.Sprintf("PROD-%d", i+1),
			Name:      fmt.Sprintf("Test Product %d", i+1),
			Price:     decimal.NewFromInt(int64((i + 1) * 100)),
		}
	}
	return products
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

func (suite *OrderRepoTestSuite) TestUpdateOrdersBatch() {
	// 創建測試用戶和產品
	user := suite.createTestUser()
	products := suite.createTestProducts(3)

	// 創建初始訂單
	initialOrders := []*model.Order{
		{
			OrderID:   "ORDER-1",
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(100),
			State:     1,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					OrderID:   "ORDER-1",
					ProductID: products[0].ProductID,
					Quantity:  1,
				},
			},
		},
		{
			OrderID:   "ORDER-2",
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(200),
			State:     1,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					OrderID:   "ORDER-2",
					ProductID: products[1].ProductID,
					Quantity:  2,
				},
			},
		},
	}

	// 先創建初始訂單
	for _, order := range initialOrders {
		err := suite.orderRepo.CreateOrder(order)
		suite.Require().NoError(err)
	}

	// 準備更新資料
	updatedOrders := []*model.Order{
		{
			OrderID:   "ORDER-1",
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(150),
			State:     2,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					OrderID:   "ORDER-1",
					ProductID: products[0].ProductID,
					Quantity:  1,
				},
				{
					OrderID:   "ORDER-1",
					ProductID: products[2].ProductID, // 新增項目
					Quantity:  1,
				},
			},
		},
		{
			OrderID:   "ORDER-2",
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(300),
			State:     2,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					OrderID:   "ORDER-2",
					ProductID: products[1].ProductID,
					Quantity:  3,
				},
			},
		},
	}

	// 執行批次更新
	result, err := suite.orderRepo.UpdateOrdersBatch(updatedOrders)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// 驗證更新結果
	suite.Equal(2, result.TotalCount, "總數應該為2")
	suite.Equal(2, result.SuccessCount, "成功數應該為2")
	suite.Equal(0, result.FailCount, "失敗數應該為0")
	suite.Len(result.SuccessOrders, 2, "成功訂單數應該為2")
	suite.Empty(result.FailedOrders, "不應該有失敗的訂單")

	// 驗證訂單更新
	for _, updatedOrder := range updatedOrders {
		order, err := suite.orderRepo.GetOrderByID(updatedOrder.OrderID)
		suite.Require().NoError(err)
		suite.Require().NotNil(order)

		// 驗證基本資訊
		suite.Equal(updatedOrder.UserID, order.UserID)
		suite.True(updatedOrder.Amount.Equal(order.Amount))
		suite.Equal(updatedOrder.State, order.State)

		// 驗證訂單項目
		suite.Len(order.OrderItems, len(updatedOrder.OrderItems))

		// 建立訂單項目映射以便比較
		expectedItems := make(map[string]model.OrderItem)
		actualItems := make(map[string]model.OrderItem)

		for _, item := range updatedOrder.OrderItems {
			expectedItems[item.ProductID] = item
		}
		for _, item := range order.OrderItems {
			actualItems[item.ProductID] = item
		}

		// 比較每個項目
		for productID, expectedItem := range expectedItems {
			actualItem, exists := actualItems[productID]
			suite.True(exists)
			suite.Equal(expectedItem.Quantity, actualItem.Quantity)
		}
	}
}

// 測試部分失敗的情況
func (suite *OrderRepoTestSuite) TestUpdateOrdersBatch_PartialFailure() {
	// 創建測試用戶和產品
	user := suite.createTestUser()
	products := suite.createTestProducts(2)

	// 創建一個有效訂單
	validOrder := &model.Order{
		OrderID:   "ORDER-1",
		UserID:    user.UserID,
		Amount:    decimal.NewFromInt(100),
		State:     1,
		OrderDate: time.Now(),
		OrderItems: []model.OrderItem{
			{
				OrderID:   "ORDER-1",
				ProductID: products[0].ProductID,
				Quantity:  1,
			},
		},
	}
	err := suite.orderRepo.CreateOrder(validOrder)
	suite.Require().NoError(err)

	// 準備更新資料（包含一個不存在的訂單）
	updatedOrders := []*model.Order{
		{
			OrderID: "ORDER-1",
			UserID:  user.UserID,
			Amount:  decimal.NewFromInt(150),
			State:   2,
			OrderItems: []model.OrderItem{
				{
					OrderID:   "ORDER-1",
					ProductID: products[0].ProductID,
					Quantity:  2,
				},
			},
		},
		{
			OrderID: "NON-EXISTENT", // 不存在的訂單
			UserID:  user.UserID,
			Amount:  decimal.NewFromInt(200),
			State:   1,
			OrderItems: []model.OrderItem{
				{
					OrderID:   "NON-EXISTENT",
					ProductID: products[1].ProductID,
					Quantity:  1,
				},
			},
		},
	}

	// 執行批次更新
	result, err := suite.orderRepo.UpdateOrdersBatch(updatedOrders)
	suite.Require().NoError(err) // 整體操作不應該返回錯誤
	suite.Require().NotNil(result)

	// 驗證部分成功的結果
	suite.Equal(2, result.TotalCount, "總數應該為2")
	suite.Equal(1, result.SuccessCount, "成功數應該為1")
	suite.Equal(1, result.FailCount, "失敗數應該為1")
	suite.Len(result.SuccessOrders, 1, "成功訂單數應該為1")
	suite.Len(result.FailedOrders, 1, "失敗訂單數應該為1")
	suite.Contains(result.FailedOrders, "NON-EXISTENT", "應該包含不存在訂單的錯誤")

	// 驗證成功更新的訂單
	updatedOrder, err := suite.orderRepo.GetOrderByID("ORDER-1")
	suite.Require().NoError(err)
	suite.Equal(decimal.NewFromInt(150), updatedOrder.Amount)
	suite.Equal(2, updatedOrder.State)
	suite.Len(updatedOrder.OrderItems, 1)
	suite.Equal(2, updatedOrder.OrderItems[0].Quantity)
}
