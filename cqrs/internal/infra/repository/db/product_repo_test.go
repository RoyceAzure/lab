package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
	"github.com/shopspring/decimal"
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
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(suite.T(), err)

	dbDao := NewDbDao(db)
	productRepo := NewProductRepo(dbDao, nil)
	suite.db = db
	suite.productRepo = productRepo
}

// SetupTest 在每個測試前執行
func (suite *ProductRepoTestSuite) SetupTest() {
	// 清空資料表
	suite.db.Exec("DELETE FROM order_items")
	suite.db.Exec("DELETE FROM products")
}

// TearDownSuite 在測試套件結束後執行
func (suite *ProductRepoTestSuite) TearDownSuite() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()
}

func (suite *ProductRepoTestSuite) TestCreateProduct() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}

	err := suite.productRepo.CreateProduct(product)

	require.NoError(suite.T(), err)
	require.NotZero(suite.T(), product.ProductID)
	require.False(suite.T(), product.CreatedAt.IsZero())
}

func (suite *ProductRepoTestSuite) TestCreateProduct_DuplicateCode() {
	product1 := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product 1",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description 1",
	}

	product2 := &model.Product{
		Code:        "TEST001", // 重複的 code
		Name:        "Test Product 2",
		Price:       decimal.NewFromFloat(200.0),
		Stock:       20,
		Category:    "Test",
		Description: "Test Description 2",
	}

	err1 := suite.productRepo.CreateProduct(product1)
	err2 := suite.productRepo.CreateProduct(product2)

	require.NoError(suite.T(), err1)
	require.Error(suite.T(), err2) // 應該會失敗
}

func (suite *ProductRepoTestSuite) TestGetProductByID() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	foundProduct, err := suite.productRepo.GetProductByID(context.Background(), product.ProductID)

	require.NoError(suite.T(), err)
	require.Equal(suite.T(), product.Name, foundProduct.Name)
	require.True(suite.T(), product.Price.Equal(foundProduct.Price))
}

func (suite *ProductRepoTestSuite) TestGetProductByCode() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	foundProduct, err := suite.productRepo.GetProductByCode("TEST001")

	require.NoError(suite.T(), err)
	require.Equal(suite.T(), product.Name, foundProduct.Name)
	require.True(suite.T(), product.Price.Equal(foundProduct.Price))
}

func (suite *ProductRepoTestSuite) TestGetAllProducts() {
	// 創建多個商品
	products := []*model.Product{
		{
			Code:        "TEST001",
			Name:        "Test Product 1",
			Price:       decimal.NewFromFloat(100.0),
			Stock:       10,
			Category:    "Test",
			Description: "Test Description 1",
		},
		{
			Code:        "TEST002",
			Name:        "Test Product 2",
			Price:       decimal.NewFromFloat(200.0),
			Stock:       20,
			Category:    "Test",
			Description: "Test Description 2",
		},
	}

	for _, product := range products {
		suite.productRepo.CreateProduct(product)
	}

	allProducts, err := suite.productRepo.GetAllProducts()

	require.NoError(suite.T(), err)
	require.Len(suite.T(), allProducts, 2)
}

func (suite *ProductRepoTestSuite) TestUpdateProduct() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	// 更新商品
	product.Name = "Updated Product"
	product.Price = decimal.NewFromFloat(150.0)
	err := suite.productRepo.UpdateProduct(product)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedProduct, _ := suite.productRepo.GetProductByID(context.Background(), product.ProductID)
	require.Equal(suite.T(), "Updated Product", updatedProduct.Name)
	require.True(suite.T(), decimal.NewFromFloat(150.0).Equal(updatedProduct.Price))
}

func (suite *ProductRepoTestSuite) TestUpdateProductFields() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	updates := map[string]interface{}{
		"name":  "Updated Product",
		"price": decimal.NewFromFloat(150.0),
	}

	err := suite.productRepo.UpdateProductFields(product.ProductID, updates)
	require.NoError(suite.T(), err)

	// 驗證更新
	updatedProduct, _ := suite.productRepo.GetProductByID(context.Background(), product.ProductID)
	require.Equal(suite.T(), "Updated Product", updatedProduct.Name)
	require.True(suite.T(), decimal.NewFromFloat(150.0).Equal(updatedProduct.Price))
}

func (suite *ProductRepoTestSuite) TestDeleteProduct() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	err := suite.productRepo.DeleteProduct(product.ProductID)
	require.NoError(suite.T(), err)

	// 驗證軟刪除
	foundProduct, err := suite.productRepo.GetProductByID(context.Background(), product.ProductID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundProduct)
}

func (suite *ProductRepoTestSuite) TestHardDeleteProduct() {
	product := &model.Product{
		Code:        "TEST001",
		Name:        "Test Product",
		Price:       decimal.NewFromFloat(100.0),
		Stock:       10,
		Category:    "Test",
		Description: "Test Description",
	}
	suite.productRepo.CreateProduct(product)

	err := suite.productRepo.HardDeleteProduct(product.ProductID)
	require.NoError(suite.T(), err)

	// 驗證硬刪除
	foundProduct, err := suite.productRepo.GetProductByID(context.Background(), product.ProductID)
	require.Error(suite.T(), err)
	require.Nil(suite.T(), foundProduct)
}

func (suite *ProductRepoTestSuite) TestGetProductsPaginated() {
	// 創建 25 個商品
	for i := 1; i <= 25; i++ {
		product := &model.Product{
			Code:        fmt.Sprintf("TEST%03d", i),
			Name:        fmt.Sprintf("Test Product %d", i),
			Price:       decimal.NewFromFloat(float64(i * 100)),
			Stock:       uint(i * 10),
			Category:    "Test",
			Description: fmt.Sprintf("Test Description %d", i),
		}
		suite.productRepo.CreateProduct(product)
	}

	// 測試第一頁，每頁 10 筆
	products, total, err := suite.productRepo.GetProductsPaginated(1, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), products, 10)
	require.Equal(suite.T(), int64(25), total)

	// 測試第三頁，每頁 10 筆
	products, total, err = suite.productRepo.GetProductsPaginated(3, 10)
	require.NoError(suite.T(), err)
	require.Len(suite.T(), products, 5) // 第三頁只有 5 筆
	require.Equal(suite.T(), int64(25), total)
}

// 執行測試套件
func TestProductServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepoTestSuite))
}

// 單獨的單元測試範例
func TestNewProductRepo(t *testing.T) {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(t, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao, nil)

	require.NotNil(t, repo)
	require.Equal(t, dbDao, repo.db)
}

// 基準測試範例
func BenchmarkCreateProduct(b *testing.B) {
	db, err := GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(b, err)

	dbDao := NewDbDao(db)
	repo := NewProductRepo(dbDao, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		product := &model.Product{
			Code:        fmt.Sprintf("TEST%03d", i),
			Name:        fmt.Sprintf("Test Product %d", i),
			Price:       decimal.NewFromFloat(float64(i * 100)),
			Stock:       uint(i * 10),
			Category:    "Test",
			Description: fmt.Sprintf("Test Description %d", i),
		}
		repo.CreateProduct(product)
	}
}
