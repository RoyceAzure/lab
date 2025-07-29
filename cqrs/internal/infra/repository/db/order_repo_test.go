package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/cqrs/internal/pkg/util"
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

func (suite *OrderRepoTestSuite) TearDownTest() {
	suite.db.Exec("DELETE FROM order_items")
	suite.db.Exec("DELETE FROM orders")
	suite.db.Exec("DELETE FROM users")
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

func (suite *OrderRepoTestSuite) TestUpdateOrdersBatch() {
	// 創建測試用戶和產品
	user := suite.createTestUser()
	products := suite.createTestProducts(3)

	// 創建初始訂單
	initialOrders := []*model.Order{
		{
			OrderID:   util.GenerateOrderID(),
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(100),
			State:     1,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					ProductID: products[0].ProductID,
					Quantity:  1,
				},
			},
		},
		{
			OrderID:   util.GenerateOrderID(),
			UserID:    user.UserID,
			Amount:    decimal.NewFromInt(200),
			State:     1,
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					ProductID: products[1].ProductID,
					Quantity:  2,
				},
			},
		},
	}

	// 先創建初始訂單
	for _, order := range initialOrders {
		err := suite.orderRepo.CreateOrder(context.Background(), order)
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
	result, err := suite.orderRepo.UpdateOrdersBatch(context.Background(), updatedOrders)
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
		order, err := suite.orderRepo.GetOrderByID(context.Background(), updatedOrder.OrderID)
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

func (suite *OrderRepoTestSuite) TestCreateOrderWithItems() {
	t := suite.T()
	user := suite.createTestUser()
	products := suite.createTestProducts(2)

	// 創建訂單和訂單項目
	order := &model.Order{
		OrderID:   util.GenerateOrderID(),
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(300.0),
		OrderDate: time.Now(),
		OrderItems: []model.OrderItem{
			{
				ProductID: products[0].ProductID,
				Quantity:  2,
			},
			{
				ProductID: products[1].ProductID,
				Quantity:  1,
			},
		},
	}

	// 創建訂單
	err := suite.orderRepo.CreateOrder(context.Background(), order)
	require.NoError(t, err)
	require.NotEmpty(t, order.OrderID)

	// 驗證訂單和訂單項目都被正確創建
	foundOrder, err := suite.orderRepo.GetOrderByID(context.Background(), order.OrderID)
	require.NoError(t, err)
	require.NotNil(t, foundOrder)
	require.Equal(t, user.UserID, foundOrder.UserID)
	require.True(t, order.Amount.Equal(foundOrder.Amount))

	// 驗證訂單項目
	require.Len(t, foundOrder.OrderItems, 2)
	for i, item := range foundOrder.OrderItems {
		require.Equal(t, order.OrderItems[i].ProductID, item.ProductID)
		require.Equal(t, order.OrderItems[i].Quantity, item.Quantity)
		require.Equal(t, order.OrderID, item.OrderID)
	}
}

func (suite *OrderRepoTestSuite) TestHardDeleteOrderWithItems() {
	t := suite.T()
	user := suite.createTestUser()
	products := suite.createTestProducts(2)

	// 創建訂單和訂單項目
	order := &model.Order{
		OrderID:   util.GenerateOrderID(),
		UserID:    user.UserID,
		Amount:    decimal.NewFromFloat(300.0),
		OrderDate: time.Now(),
		OrderItems: []model.OrderItem{
			{
				ProductID: products[0].ProductID,
				Quantity:  2,
			},
			{
				ProductID: products[1].ProductID,
				Quantity:  1,
			},
		},
	}

	// 創建訂單
	err := suite.orderRepo.CreateOrder(context.Background(), order)
	require.NoError(t, err)

	// 硬刪除訂單
	err = suite.orderRepo.HardDeleteOrder(context.Background(), order.OrderID)
	require.NoError(t, err)

	// 驗證訂單被硬刪除
	foundOrder, err := suite.orderRepo.GetOrderByID(context.Background(), order.OrderID)
	require.Error(t, err)
	require.Nil(t, foundOrder)

	// 驗證訂單項目也被硬刪除（使用 Unscoped 也找不到）
	var count int64
	err = suite.db.Unscoped().Model(&model.OrderItem{}).Where("order_id = ?", order.OrderID).Count(&count).Error
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func (suite *OrderRepoTestSuite) TestGetOrdersByUserIDWithItems() {
	t := suite.T()
	user := suite.createTestUser()
	products := suite.createTestProducts(2)

	// 創建多個訂單
	orders := []*model.Order{
		{
			OrderID:   util.GenerateOrderID(),
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(300.0),
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					ProductID: products[0].ProductID,
					Quantity:  2,
				},
				{
					ProductID: products[1].ProductID,
					Quantity:  1,
				},
			},
		},
		{
			OrderID:   util.GenerateOrderID(),
			UserID:    user.UserID,
			Amount:    decimal.NewFromFloat(200.0),
			OrderDate: time.Now(),
			OrderItems: []model.OrderItem{
				{
					ProductID: products[0].ProductID,
					Quantity:  1,
				},
			},
		},
	}

	// 創建訂單
	for _, order := range orders {
		err := suite.orderRepo.CreateOrder(context.Background(), order)
		require.NoError(t, err)
	}

	// 獲取用戶的所有訂單
	foundOrders, err := suite.orderRepo.GetOrdersByUserID(context.Background(), user.UserID)
	require.NoError(t, err)
	require.Len(t, foundOrders, 2)

	// 驗證每個訂單的訂單項目
	for i, foundOrder := range foundOrders {
		require.Equal(t, orders[i].UserID, foundOrder.UserID)
		require.True(t, orders[i].Amount.Equal(foundOrder.Amount))
		require.Len(t, foundOrder.OrderItems, len(orders[i].OrderItems))

		// 驗證訂單項目
		for j, item := range foundOrder.OrderItems {
			require.Equal(t, orders[i].OrderItems[j].ProductID, item.ProductID)
			require.Equal(t, orders[i].OrderItems[j].Quantity, item.Quantity)
			require.Equal(t, foundOrder.OrderID, item.OrderID)
		}
	}
}

func TestOrderRepoSuite(t *testing.T) {
	suite.Run(t, new(OrderRepoTestSuite))
}
