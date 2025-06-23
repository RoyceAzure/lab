package db

import (
	"fmt"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/config"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type UserRepoTestSuite struct {
	suite.Suite
	db       *gorm.DB
	userRepo *UserRepo
}

// SetupSuite 在測試套件開始前執行
func (suite *UserRepoTestSuite) SetupSuite() {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(suite.T(), err)
	dbDao := NewDbDao(db)
	userRepo := NewUserRepo(dbDao)
	suite.db = db
	suite.userRepo = userRepo
}

// SetupTest 在每個測試前執行
func (suite *UserRepoTestSuite) SetupTest() {
	// 清空資料表
	suite.db.Exec("DELETE FROM users")
	suite.db.Exec("DELETE FROM orders")
}

// TearDownSuite 在測試套件結束後執行
func (suite *UserRepoTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *UserRepoTestSuite) TestCreateUser() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}

	err := suite.userRepo.CreateUser(user)

	require.NoError(suite.T(), err)
	require.NotZero(suite.T(), user.UserID)
	require.False(suite.T(), user.CreatedAt.IsZero())
}

func (suite *UserRepoTestSuite) TestCreateUser_DuplicateEmail() {
	user1 := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}

	user2 := &model.User{
		UserName:    "Jane Doe",
		UserEmail:   "john@example.com", // 重複的 email
		UserPhone:   "0987654321",
		UserAddress: "456 Oak St",
	}

	err1 := suite.userRepo.CreateUser(user1)
	err2 := suite.userRepo.CreateUser(user2)

	require.NoError(suite.T(), err1)
	require.Error(suite.T(), err2) // 應該會失敗
}

func (suite *UserRepoTestSuite) TestGetUserByID() {
	// 先創建一個用戶
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	// 測試查詢
	foundUser, err := suite.userRepo.GetUserByID(user.UserID)

	require.NoError(suite.T(), err)
	require.Equal(suite.T(), user.UserName, foundUser.UserName)
	require.Equal(suite.T(), user.UserEmail, foundUser.UserEmail)
}

func (suite *UserRepoTestSuite) TestGetUserByID_NotFound() {
	foundUser, err := suite.userRepo.GetUserByID(999)

	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundUser)
}

func (suite *UserRepoTestSuite) TestGetUserByEmail() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	foundUser, err := suite.userRepo.GetUserByEmail("john@example.com")

	require.NoError(suite.T(), err)
	require.Equal(suite.T(), user.UserName, foundUser.UserName)
}

func (suite *UserRepoTestSuite) TestGetAllUsers() {
	// 創建多個用戶
	users := []*model.User{
		{
			UserName:    "John Doe",
			UserEmail:   "john@example.com",
			UserPhone:   "1234567890",
			UserAddress: "123 Main St",
		},
		{
			UserName:    "Jane Smith",
			UserEmail:   "jane@example.com",
			UserPhone:   "0987654321",
			UserAddress: "456 Oak St",
		},
	}

	for _, user := range users {
		suite.userRepo.CreateUser(user)
	}

	allUsers, err := suite.userRepo.GetAllUsers()

	require.NoError(suite.T(), err)
	require.Len(suite.T(), allUsers, 2)
}

func (suite *UserRepoTestSuite) TestUpdateUser() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	// 更新用戶資訊
	user.UserName = "John Updated"
	user.UserAddress = "456 Updated St"

	err := suite.userRepo.UpdateUser(user)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedUser, _ := suite.userRepo.GetUserByID(user.UserID)
	require.Equal(suite.T(), "John Updated", updatedUser.UserName)
	require.Equal(suite.T(), "456 Updated St", updatedUser.UserAddress)
}

func (suite *UserRepoTestSuite) TestUpdateUserFields() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	updates := map[string]interface{}{
		"user_name":    "Jane Doe",
		"user_address": "789 New St",
	}

	err := suite.userRepo.PatchUserFields(user.UserID, updates)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedUser, _ := suite.userRepo.GetUserByID(user.UserID)
	require.Equal(suite.T(), "Jane Doe", updatedUser.UserName)
	require.Equal(suite.T(), "789 New St", updatedUser.UserAddress)
	require.Equal(suite.T(), "john@example.com", updatedUser.UserEmail) // 未更新的欄位應該保持不變
}

func (suite *UserRepoTestSuite) TestDeleteUser() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	err := suite.userRepo.DeleteUser(user.UserID)
	require.NoError(suite.T(), err)

	// 驗證軟刪除 - 用戶應該查不到
	foundUser, err := suite.userRepo.GetUserByID(user.UserID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundUser)
}

func (suite *UserRepoTestSuite) TestHardDeleteUser() {
	user := &model.User{
		UserName:    "John Doe",
		UserEmail:   "john@example.com",
		UserPhone:   "1234567890",
		UserAddress: "123 Main St",
	}
	suite.userRepo.CreateUser(user)

	err := suite.userRepo.HardDeleteUser(user.UserID)
	require.NoError(suite.T(), err)

	// 驗證硬刪除 - 用戶應該查不到
	foundUser, err := suite.userRepo.GetUserByID(user.UserID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundUser)
}

func (suite *UserRepoTestSuite) TestGetUsersPaginated() {
	// 創建 25 個用戶
	for i := 1; i <= 25; i++ {
		user := &model.User{
			UserName:    fmt.Sprintf("User %d", i),
			UserEmail:   fmt.Sprintf("user%d@example.com", i),
			UserPhone:   fmt.Sprintf("123456789%d", i),
			UserAddress: fmt.Sprintf("%d Main St", i),
		}
		suite.userRepo.CreateUser(user)
	}

	// 測試第一頁，每頁 10 筆
	users, total, err := suite.userRepo.GetUsersPaginated(1, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), users, 10)
	require.Equal(suite.T(), int64(25), total)

	// 測試第三頁，每頁 10 筆
	users, total, err = suite.userRepo.GetUsersPaginated(3, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), users, 5) // 第三頁只有 5 筆
	require.Equal(suite.T(), int64(25), total)
}

func (suite *UserRepoTestSuite) TestGetUsersPaginated_EmptyResult() {
	users, total, err := suite.userRepo.GetUsersPaginated(1, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), users, 0)
	require.Equal(suite.T(), int64(0), total)
}

// 執行測試套件
func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepoTestSuite))
}

// 單獨的單元測試範例
func TestNewUserRepo(t *testing.T) {
	// 獲取資料庫配置
	cfg, err := config.GetPGConfig()
	require.NoError(t, err)

	// 連接到資料庫
	db, err := GetDbConn(cfg.DbName, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPas)
	require.NoError(t, err)

	dbDao := NewDbDao(db)
	repo := NewUserRepo(dbDao)

	require.NotNil(t, repo)
	require.Equal(t, dbDao, repo.dbDao)
}

// 基準測試範例
func BenchmarkCreateUser(b *testing.B) {
	// 獲取資料庫配置
	cfg, err := config.GetPGConfig()
	require.NoError(b, err)

	// 連接到資料庫
	db, err := GetDbConn(cfg.DbName, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPas)
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewUserRepo(dbDao)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &model.User{
			UserName:    fmt.Sprintf("User %d", i),
			UserEmail:   fmt.Sprintf("user%d@example.com", i),
			UserPhone:   fmt.Sprintf("123456789%d", i),
			UserAddress: fmt.Sprintf("%d Main St", i),
		}
		repo.CreateUser(user)
	}
}
