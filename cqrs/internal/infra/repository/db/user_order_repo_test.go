package db

import (
	"context"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserOrderRepoTestSuite struct {
	suite.Suite
	dbDao         *DbDao
	userOrderRepo *UserOrderRepo
	userRepo      *UserRepo
	testUser      *model.User
}

func (suite *UserOrderRepoTestSuite) SetupSuite() {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	assert.NoError(suite.T(), err)
	suite.dbDao = NewDbDao(db)
	err = suite.dbDao.InitMigrate()
	assert.NoError(suite.T(), err)
	suite.userOrderRepo = NewUserOrderRepo(suite.dbDao)
	suite.userRepo = NewUserRepo(suite.dbDao)

}

func (suite *UserOrderRepoTestSuite) TearDownSuite() {

}

func (suite *UserOrderRepoTestSuite) SetupTest() {
	var err error
	suite.testUser, err = suite.userRepo.CreateUser(context.Background(), &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	})
	assert.NoError(suite.T(), err)
}

func (suite *UserOrderRepoTestSuite) TearDownTest() {
	suite.dbDao.Exec("DELETE FROM user_orders")
	suite.dbDao.Exec("DELETE FROM users")
}

func TestUserOrderRepoTestSuite(t *testing.T) {
	suite.Run(t, new(UserOrderRepoTestSuite))
}

func (suite *UserOrderRepoTestSuite) TestCreateUserOrder() {
	t := suite.T()

	// 準備測試資料
	testUserOrder := &model.UserOrder{
		UserID:  suite.testUser.UserID,
		OrderID: "test-order-1",
	}

	// 執行測試
	createdUserOrder, err := suite.userOrderRepo.CreateUserOrder(testUserOrder)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, createdUserOrder)
	assert.NotZero(t, createdUserOrder.ID)
	assert.Equal(t, testUserOrder.UserID, createdUserOrder.UserID)
	assert.Equal(t, testUserOrder.OrderID, createdUserOrder.OrderID)
}

func (suite *UserOrderRepoTestSuite) TestGetUserOrderByID() {
	t := suite.T()

	// 準備測試資料
	testUserOrder := &model.UserOrder{
		UserID:  suite.testUser.UserID,
		OrderID: "test-order-1",
	}
	createdUserOrder, err := suite.userOrderRepo.CreateUserOrder(testUserOrder)
	assert.NoError(t, err)

	// 執行測試
	foundUserOrder, err := suite.userOrderRepo.GetUserOrderByID(createdUserOrder.ID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, foundUserOrder)
	assert.Equal(t, createdUserOrder.ID, foundUserOrder.ID)
	assert.Equal(t, createdUserOrder.UserID, foundUserOrder.UserID)
	assert.Equal(t, createdUserOrder.OrderID, foundUserOrder.OrderID)
}

func (suite *UserOrderRepoTestSuite) TestListUserOrders() {
	t := suite.T()

	// 準備測試資料
	testUserOrders := []model.UserOrder{
		{UserID: suite.testUser.UserID, OrderID: "test-order-1"},
		{UserID: suite.testUser.UserID, OrderID: "test-order-2"},
		{UserID: suite.testUser.UserID, OrderID: "test-order-3"},
	}

	for _, order := range testUserOrders {
		_, err := suite.userOrderRepo.CreateUserOrder(&order)
		assert.NoError(t, err)
	}

	// 執行測試
	foundUserOrders, err := suite.userOrderRepo.ListUserOrders()

	// 驗證結果
	assert.NoError(t, err)
	assert.Len(t, foundUserOrders, len(testUserOrders))
}

func (suite *UserOrderRepoTestSuite) TestListUserOrdersByUserID() {
	t := suite.T()

	// 準備測試資料
	testUserOrders := []model.UserOrder{
		{UserID: suite.testUser.UserID, OrderID: "test-order-1"},
		{UserID: suite.testUser.UserID, OrderID: "test-order-2"},
		{UserID: suite.testUser.UserID, OrderID: "test-order-3"},
	}

	for _, order := range testUserOrders {
		_, err := suite.userOrderRepo.CreateUserOrder(&order)
		assert.NoError(t, err)
	}

	// 執行測試
	foundUserOrders, err := suite.userOrderRepo.ListUserOrdersByUserID(suite.testUser.UserID)

	// 驗證結果
	assert.NoError(t, err)
	assert.Len(t, foundUserOrders, len(testUserOrders))
	for _, order := range foundUserOrders {
		assert.Equal(t, suite.testUser.UserID, order.UserID)
	}
}

func (suite *UserOrderRepoTestSuite) TestHardDeleteUserOrder() {
	t := suite.T()

	// 準備測試資料
	testUserOrder := &model.UserOrder{
		UserID:  suite.testUser.UserID,
		OrderID: "test-order-1",
	}
	createdUserOrder, err := suite.userOrderRepo.CreateUserOrder(testUserOrder)
	assert.NoError(t, err)

	// 執行測試
	err = suite.userOrderRepo.HardDeleteUserOrder(createdUserOrder.ID) // 使用硬刪除
	assert.NoError(t, err)

	// 驗證結果 - 使用 Unscoped 查詢確認記錄真的被刪除
	var count int64
	suite.dbDao.Unscoped().Model(&model.UserOrder{}).Where("id = ?", createdUserOrder.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}
