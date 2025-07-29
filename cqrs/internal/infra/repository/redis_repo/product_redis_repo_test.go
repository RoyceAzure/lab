package redis_repo

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/errgroup"
)

type ProductRepoTestSuite struct {
	suite.Suite
	productRepo *ProductRedisRepo
}

func (suite *ProductRepoTestSuite) SetupTest() {
	rdb := setupTestRedis()
	rdb.FlushDB(context.Background())
	suite.productRepo = NewProductRepo(rdb)
}

func TestProductRepoTestSuite(t *testing.T) {
	suite.Run(t, new(ProductRepoTestSuite))
}

func (suite *ProductRepoTestSuite) TestBasicProductStockOperations() {
	ctx := context.Background()

	// 創建商品庫存
	err := suite.productRepo.CreateProductStock(ctx, "test1", 100)
	assert.NoError(suite.T(), err)

	// 檢查庫存
	stock, err := suite.productRepo.GetProductStock(ctx, "test1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 100, stock)

	// 增加庫存
	_, err = suite.productRepo.AddProductStock(ctx, "test1", 50)
	assert.NoError(suite.T(), err)

	// 檢查增加後的庫存
	stock, err = suite.productRepo.GetProductStock(ctx, "test1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 150, stock)

	// 扣減庫存
	newStock, err := suite.productRepo.DeductProductStock(ctx, "test1", 30)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 120, newStock)
}

func (suite *ProductRepoTestSuite) TestConcurrentStockOperations() {
	// 創建兩個 context：一個用於操作，一個用於最終檢查
	opCtx, opCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer opCancel()

	// 初始化商品庫存
	initialStock := uint(1000)
	err := suite.productRepo.CreateProductStock(opCtx, "test2", initialStock)
	assert.NoError(suite.T(), err)

	// 設定併發數量
	const (
		numGoroutines = 1000
		addQuantity   = uint(5)
	)

	// 創建 errgroup，它會繼承 timeout context
	g, ctx := errgroup.WithContext(opCtx)

	// 用於計算庫存不足錯誤的次數
	insufficientCount := int32(0)

	// 啟動增加庫存的 goroutines
	for i := 0; i < numGoroutines; i++ {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				_, err := suite.productRepo.AddProductStock(ctx, "test2", addQuantity)
				if err != nil {
					return err
				}
				return nil
			}
		})
	}

	// 啟動扣減庫存的 goroutines
	for i := 0; i < numGoroutines; i++ {
		g.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				_, err := suite.productRepo.DeductProductStock(ctx, "test2", addQuantity)
				if err != nil {
					if errors.Is(err, ErrProductStockNotEnough) {
						atomic.AddInt32(&insufficientCount, 1)
						return nil
					}
					return err
				}
				return nil
			}
		})
	}

	// 等待所有 goroutine 完成
	err = g.Wait()

	// 如果是超時錯誤，輸出更多信息
	if err != nil {
		if err == context.DeadlineExceeded {
			suite.T().Logf("Test timed out: some operations did not complete in time")
		}
		suite.T().Logf("Error occurred: %v", err)
		assert.NoError(suite.T(), err)
		return
	}

	// 使用新的 context 來檢查最終結果
	checkCtx := context.Background()
	finalStock, err := suite.productRepo.GetProductStock(checkCtx, "test2")
	assert.NoError(suite.T(), err)

	// 計算期望的最終庫存：
	// 初始庫存 + 增加操作總量 - 成功的扣減操作總量
	// = initialStock + (numGoroutines * addQuantity) - ((numGoroutines - insufficientCount) * addQuantity)
	expectedStock := int(initialStock) + numGoroutines*int(addQuantity) - (numGoroutines-int(insufficientCount))*int(addQuantity)
	suite.T().Logf("Insufficient stock errors: %d", insufficientCount)
	suite.T().Logf("Expected final stock: %d", expectedStock)

	assert.Equal(suite.T(), expectedStock, finalStock)
}

func (suite *ProductRepoTestSuite) TestDeductProductStockErrors() {
	ctx := context.Background()

	// 測試商品不存在的情況
	_, err := suite.productRepo.DeductProductStock(ctx, "nonexistent", 10)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "not found")

	// 創建商品庫存
	err = suite.productRepo.CreateProductStock(ctx, "test3", 50)
	assert.NoError(suite.T(), err)

	// 測試庫存不足的情況
	_, err = suite.productRepo.DeductProductStock(ctx, "test3", 100)
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "stock not enough")
}
