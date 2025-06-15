package db

import (
	"fmt"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/config"
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ProductRepoTestSuite struct {
	suite.Suite
	db          *gorm.DB
	productRepo *ProductRepo
}

// SetupSuite 在測試套件開始前執行
func (suite *ProductRepoTestSuite) SetupSuite() {
	// 獲取資料庫配置
	cfg, err := config.GetPGConfig()
	require.NoError(suite.T(), err)

	// 連接到資料庫
	db, err := GetDbConn(cfg.DbName, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPas)
	require.NoError(suite.T(), err)

	// 初始化資料庫
	dbDao := NewDbDao(db)
	err = dbDao.InitMigrate()
	require.NoError(suite.T(), err)

	productRepo := NewProductRepo(dbDao)
	suite.db = db
	suite.productRepo = productRepo
}

// SetupTest 在每個測試前執行
func (suite *ProductRepoTestSuite) SetupTest() {
	// 清空資料表
	suite.db.Exec("DELETE FROM products")
}

func (suite *ProductRepoTestSuite) TearDownSuite() {
	// 清理測試資料（可選）
	db, err := suite.db.DB()
	require.NoError(suite.T(), err)
	db.Close()
}

func (suite *ProductRepoTestSuite) TestCreateAndGetProduct() {
	// 創建商品
	newProduct := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       "5000", // 50.00 的 cent 表示
		Stock:       100,
		Category:    "Test",
		Description: "Test Description",
	}
	err := suite.productRepo.CreateProduct(newProduct)
	require.NoError(suite.T(), err, "Failed to create product")
	require.NotZero(suite.T(), newProduct.ProductID, "Product ID should be set")

	// 根據 ID 查詢
	retrievedProduct, err := suite.productRepo.GetProductByID(newProduct.ProductID)
	require.NoError(suite.T(), err, "Failed to get product by ID")
	require.Equal(suite.T(), newProduct.Code, retrievedProduct.Code, "Product code mismatch")
}

func (suite *ProductRepoTestSuite) TestUpdateStock() {
	// 創建商品
	newProduct := &model.Product{
		Code:        "TEST002",
		Name:        "Test Product 2",
		Price:       "10000", // 100.00 的 cent 表示
		Stock:       50,
		Category:    "Test",
		Description: "Test Description 2",
	}
	err := suite.productRepo.CreateProduct(newProduct)
	require.NoError(suite.T(), err, "Failed to create product")
	require.NotZero(suite.T(), newProduct.ProductID, "Product ID should be set")

	// 更新庫存
	err = suite.productRepo.UpdateStock(newProduct.ProductID, 75)
	require.NoError(suite.T(), err, "Failed to update stock")

	// 驗證更新後的庫存
	retrievedProduct, err := suite.productRepo.GetProductByID(newProduct.ProductID)
	require.NoError(suite.T(), err, "Failed to get product by ID")
	require.Equal(suite.T(), uint(75), retrievedProduct.Stock, "Stock should be updated to 75")
}

// Benchmark 測試
func BenchmarkGetAllProducts(b *testing.B) {
	// 連接到資料庫
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao)

	// 插入測試數據
	products := make([]model.Product, 1000)
	for i := 0; i < 1000; i++ {
		products[i] = model.Product{
			Code:        fmt.Sprintf("TEST%d", i),
			Name:        fmt.Sprintf("Test Product %d", i),
			Price:       fmt.Sprintf("%d", 5000+i),
			Stock:       uint(100 + i),
			Category:    "Test",
			Description: fmt.Sprintf("Test Description %d", i),
		}
	}
	err = repo.CreateProductsBatch(products)
	if err != nil {
		b.Fatal("Failed to create test products:", err)
	}

	// 執行 Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetAllProducts()
		if err != nil {
			b.Error("Benchmark GetAllProducts failed:", err)
		}
	}
}

func BenchmarkSearchProductsByName(b *testing.B) {
	// 連接到資料庫
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao)

	// 插入測試數據
	products := make([]model.Product, 1000)
	for i := 0; i < 1000; i++ {
		products[i] = model.Product{
			Code:        fmt.Sprintf("TEST%d", i),
			Name:        fmt.Sprintf("Test Product %d", i),
			Price:       fmt.Sprintf("%d", 5000+i),
			Stock:       uint(100 + i),
			Category:    "Test",
			Description: fmt.Sprintf("Test Description %d", i),
		}
	}
	err = repo.CreateProductsBatch(products)
	if err != nil {
		b.Fatal("Failed to create test products:", err)
	}

	// 執行 Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.SearchProductsByName("Test")
		if err != nil {
			b.Error("Benchmark SearchProductsByName failed:", err)
		}
	}
	b.StopTimer()
	repo.db.Unscoped().Delete(&model.Product{}, "code LIKE ?", "TEST%")
}

func BenchmarkGetProductsPaginated(b *testing.B) {
	// 連接到資料庫
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao)

	// 插入測試數據
	products := make([]model.Product, 1000)
	for i := 0; i < 1000; i++ {
		products[i] = model.Product{
			Code:        fmt.Sprintf("TEST%d", i),
			Name:        fmt.Sprintf("Test Product %d", i),
			Price:       fmt.Sprintf("%d", 5000+i),
			Stock:       uint(100 + i),
			Category:    "Test",
			Description: fmt.Sprintf("Test Description %d", i),
		}
	}
	err = repo.CreateProductsBatch(products)
	if err != nil {
		b.Fatal("Failed to create test products:", err)
	}

	// 執行 Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := repo.GetProductsPaginated(1, 10)
		if err != nil {
			b.Error("Benchmark GetProductsPaginated failed:", err)
		}
	}
	b.StopTimer()
	repo.db.Unscoped().Delete(&model.Product{}, "code LIKE ?", "TEST%")
}

func BenchmarkCreateProductsBatch(b *testing.B) {
	// 連接到資料庫
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao)

	// 準備測試數據
	products := make([]model.Product, 100)
	for i := 0; i < 100; i++ {
		products[i] = model.Product{
			Code:        fmt.Sprintf("BATCH%d", i),
			Name:        fmt.Sprintf("Batch Product %d", i),
			Price:       fmt.Sprintf("%d", 5000+i),
			Stock:       uint(100 + i),
			Category:    "Batch",
			Description: fmt.Sprintf("Batch Description %d", i),
		}
	}

	// 執行 Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.CreateProductsBatch(products)
		if err != nil {
			b.Error("Benchmark CreateProductsBatch failed:", err)
		}
		// 清理數據以避免重複插入衝突
		err = repo.db.Unscoped().Delete(&model.Product{}, "code LIKE ?", "BATCH%").Error
		if err != nil {
			b.Error("Failed to clean up batch data:", err)
		}
	}
}

func TestProductRepoSuite(t *testing.T) {
	suite.Run(t, new(ProductRepoTestSuite))
}
