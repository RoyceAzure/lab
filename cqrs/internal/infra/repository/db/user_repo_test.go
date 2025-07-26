package db

import (
	"context"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserRepoTestSuite struct {
	suite.Suite
	dbDao    *DbDao
	userRepo *UserRepo
	ctx      context.Context
}

func (suite *UserRepoTestSuite) SetupSuite() {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	assert.NoError(suite.T(), err)
	suite.dbDao = NewDbDao(db)
	err = suite.dbDao.InitMigrate()
	assert.NoError(suite.T(), err)
	suite.userRepo = NewUserRepo(suite.dbDao)
	suite.ctx = context.Background()
}

func (suite *UserRepoTestSuite) TearDownSuite() {
	// GORM v2 會自動管理連接池，不需要手動關閉
}

func (suite *UserRepoTestSuite) SetupTest() {
	// 清空測試資料
	suite.dbDao.Exec("DELETE FROM users")
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, new(UserRepoTestSuite))
}

func (suite *UserRepoTestSuite) TestCreateUser() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}

	// 執行測試
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.NotZero(t, createdUser.UserID)
	assert.Equal(t, testUser.UserName, createdUser.UserName)
	assert.Equal(t, testUser.UserEmail, createdUser.UserEmail)
	assert.Equal(t, testUser.UserPhone, createdUser.UserPhone)
	assert.Equal(t, testUser.UserAddress, createdUser.UserAddress)
}

func (suite *UserRepoTestSuite) TestGetUserByID() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)
	assert.NoError(t, err)

	// 執行測試
	foundUser, err := suite.userRepo.GetUserByID(suite.ctx, createdUser.UserID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, createdUser.UserID, foundUser.UserID)
	assert.Equal(t, createdUser.UserName, foundUser.UserName)
	assert.Equal(t, createdUser.UserEmail, foundUser.UserEmail)
	assert.Equal(t, createdUser.UserPhone, foundUser.UserPhone)
	assert.Equal(t, createdUser.UserAddress, foundUser.UserAddress)
}

func (suite *UserRepoTestSuite) TestGetUserByEmail() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)
	assert.NoError(t, err)

	// 執行測試
	foundUser, err := suite.userRepo.GetUserByEmail(suite.ctx, createdUser.UserEmail)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, foundUser)
	assert.Equal(t, createdUser.UserID, foundUser.UserID)
	assert.Equal(t, createdUser.UserEmail, foundUser.UserEmail)
	assert.Equal(t, createdUser.UserPhone, foundUser.UserPhone)
	assert.Equal(t, createdUser.UserAddress, foundUser.UserAddress)
}

func (suite *UserRepoTestSuite) TestGetAllUsers() {
	t := suite.T()

	// 準備測試資料
	testUsers := []*model.User{
		{
			UserName:    "User 1",
			UserEmail:   "user1@example.com",
			UserPhone:   "1234567890",
			UserAddress: "Address 1",
		},
		{
			UserName:    "User 2",
			UserEmail:   "user2@example.com",
			UserPhone:   "0987654321",
			UserAddress: "Address 2",
		},
	}

	for _, user := range testUsers {
		_, err := suite.userRepo.CreateUser(suite.ctx, user)
		assert.NoError(t, err)
	}

	// 執行測試
	foundUsers, err := suite.userRepo.GetAllUsers(suite.ctx)

	// 驗證結果
	assert.NoError(t, err)
	assert.Len(t, foundUsers, len(testUsers))
}

func (suite *UserRepoTestSuite) TestUpdateUser() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)
	assert.NoError(t, err)

	// 修改資料
	createdUser.UserName = "Updated Name"
	createdUser.UserEmail = "updated@example.com"
	createdUser.UserPhone = "9876543210"
	createdUser.UserAddress = "Updated Address"

	// 執行測試
	err = suite.userRepo.UpdateUser(suite.ctx, createdUser)
	assert.NoError(t, err)

	// 驗證結果
	updatedUser, err := suite.userRepo.GetUserByID(suite.ctx, createdUser.UserID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedUser.UserName)
	assert.Equal(t, "updated@example.com", updatedUser.UserEmail)
	assert.Equal(t, "9876543210", updatedUser.UserPhone)
	assert.Equal(t, "Updated Address", updatedUser.UserAddress)
}

func (suite *UserRepoTestSuite) TestDeleteUser() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)
	assert.NoError(t, err)

	// 執行測試
	err = suite.userRepo.DeleteUser(suite.ctx, createdUser.UserID)
	assert.NoError(t, err)

	// 驗證結果
	deletedUser, err := suite.userRepo.GetUserByID(suite.ctx, createdUser.UserID)
	assert.Error(t, err) // 應該返回錯誤，因為用戶已被刪除
	assert.Nil(t, deletedUser)
}

func (suite *UserRepoTestSuite) TestHardDeleteUser() {
	t := suite.T()

	// 準備測試資料
	testUser := &model.User{
		UserName:    "Test User",
		UserEmail:   "test@example.com",
		UserPhone:   "1234567890",
		UserAddress: "Test Address",
	}
	createdUser, err := suite.userRepo.CreateUser(suite.ctx, testUser)
	assert.NoError(t, err)

	// 執行測試
	err = suite.userRepo.HardDeleteUser(suite.ctx, createdUser.UserID)
	assert.NoError(t, err)

	// 驗證結果 - 使用 Unscoped 查詢確認記錄真的被刪除
	var count int64
	suite.dbDao.Unscoped().Model(&model.User{}).Where("user_id = ?", createdUser.UserID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func (suite *UserRepoTestSuite) TestGetUserByID_NotFound() {
	t := suite.T()

	// 測試不存在的用戶ID
	user, err := suite.userRepo.GetUserByID(suite.ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, user)
}

func (suite *UserRepoTestSuite) TestGetUserByEmail_NotFound() {
	t := suite.T()

	// 測試不存在的用戶郵箱
	user, err := suite.userRepo.GetUserByEmail(suite.ctx, "notexist@example.com")
	assert.Error(t, err)
	assert.Nil(t, user)
}
