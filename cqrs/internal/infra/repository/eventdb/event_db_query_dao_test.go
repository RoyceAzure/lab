package eventdb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

var testOrderID = "124"
var testProjectionName = "order-projection-test3"

// EventDBQueryTestSuite 定義測試套件
// 測試可以使用同一個projectionName，但streamId要不同
type EventDBQueryTestSuite struct {
	suite.Suite
	ctx              context.Context
	client           *esdb.Client
	eventDao         *EventDao
	projectionClient *esdb.ProjectionClient
	queryDao         *EventDBQueryDao
	projectionName   string

	// 新增資料庫相關欄位
	dbDao       *db.DbDao
	userRepo    *db.UserRepo
	redisClient *redis.Client
	productRepo *redis_repo.ProductRepo

	// 測試資料
	testUsers     []*model.User
	testProducts  []*model.Product
	initialStocks map[string]uint
}

// SetupSuite 在所有測試開始前執行
func (s *EventDBQueryTestSuite) SetupSuite() {
	s.T().Log("Start SetupSuite...")
	s.ctx = context.Background()

	// 設置 EventStore 客戶端
	settings, err := esdb.ParseConnectionString("esdb://localhost:2113?tls=false&keepAliveTimeout=10000")
	s.Require().NoError(err)

	s.client, err = esdb.NewClient(settings)
	s.Require().NoError(err)

	s.eventDao = NewEventDao(s.client)

	s.projectionClient, err = esdb.NewProjectionClient(settings)
	s.Require().NoError(err)

	s.queryDao = NewEventDBQueryDao(s.projectionClient, s.client)

	// 初始化 DB 連線
	conn, err := db.GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	s.Require().NoError(err)
	s.dbDao = db.NewDbDao(conn)

	err = s.dbDao.InitMigrate()
	s.Require().NoError(err)

	// 初始化 Redis 連線
	s.redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password",
		DB:       1, // 使用測試用的DB
	})

	// 清空測試用的 Redis DB
	err = s.redisClient.FlushDB(s.ctx).Err()
	s.Require().NoError(err)

	// 初始化 repositories
	s.userRepo = db.NewUserRepo(s.dbDao)
	s.productRepo = redis_repo.NewProductRepo(s.redisClient)

	s.T().Log("End SetupSuite")
}

// cleanupProjection 清理 projection
func (s *EventDBQueryTestSuite) cleanupProjection(ctx context.Context) error {
	// 先停止 projection
	err := s.projectionClient.Disable(ctx, s.projectionName, esdb.GenericProjectionOptions{})
	if err != nil {
		// 忽略錯誤，可能 projection 不存在
		s.T().Logf("停止 projection 時發生錯誤（可能不存在）: %v", err)
	}

	// 等待 projection 停止
	time.Sleep(time.Second)

	// 刪除 projection
	err = s.queryDao.DeleteOrderProjection(ctx, s.projectionName)
	if err != nil {
		// 忽略錯誤，可能 projection 已經被刪除
		s.T().Logf("刪除 projection 時發生錯誤（可能已被刪除）: %v", err)
	}

	return nil
}

// cleanupEventStream 清理事件流
func (s *EventDBQueryTestSuite) cleanupEventStream(ctx context.Context, streamID string) error {
	// 刪除事件流
	_, err := s.client.DeleteStream(ctx, streamID, esdb.DeleteStreamOptions{})
	if err != nil {
		s.T().Logf("刪除事件流 %s 時發生錯誤（可能不存在）: %v", streamID, err)
	}
	return nil
}

// cleanupAllTestData 清理所有測試數據
func (s *EventDBQueryTestSuite) cleanupAllTestData(ctx context.Context) {
	s.T().Log("開始清理所有測試數據...")

	// 清理測試用的事件流
	testOrderIDs := []string{"123"}
	for _, orderID := range testOrderIDs {
		streamID := GenerateOrderStreamID(orderID)
		if err := s.cleanupEventStream(ctx, streamID); err != nil {
			s.T().Logf("清理事件流 %s 時發生錯誤: %v", streamID, err)
		}
	}

	// 清理整個 order category
	_, err := s.client.DeleteStream(ctx, "$ce-order", esdb.DeleteStreamOptions{})
	if err != nil {
		s.T().Logf("清理 order category 時發生錯誤: %v", err)
	}

	s.T().Log("完成清理所有測試數據")
}

// TearDownSuite 在所有測試結束後執行
func (s *EventDBQueryTestSuite) TearDownSuite() {
	s.T().Log("Start TearDownSuite...")

	// 創建一個有超時的 context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// 清理所有測試數據
	s.cleanupAllTestData(ctx)

	// 關閉 Redis 連線
	if s.redisClient != nil {
		s.redisClient.Close()
	}

	// 關閉 EventStore 連線
	if s.projectionClient != nil {
		s.projectionClient.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
	s.T().Log("End TearDownSuite")
}

// SetupTest 在每個測試開始前執行
func (s *EventDBQueryTestSuite) SetupTest() {
	s.T().Log("Start SetupTest...")
	//每次測試都創建一個新的 projection
	s.projectionName = testProjectionName
	//有重複也不理會
	s.queryDao.CreateOrderProjection(s.ctx, s.projectionName)

	// 初始化 initialStocks map
	s.initialStocks = make(map[string]uint)

	// 準備測試使用者資料
	s.testUsers = make([]*model.User, 10) // 建立10個測試使用者
	for i := 0; i < 10; i++ {
		user := &model.User{
			UserName:    fmt.Sprintf("Test User %d", i+1),
			UserEmail:   fmt.Sprintf("test%d@example.com", i+1),
			UserPhone:   fmt.Sprintf("09%d", i+1),
			UserAddress: "test address",
		}
		// 建立測試使用者並獲取創建後的用戶
		createdUser, err := s.userRepo.CreateUser(user)
		s.Require().NoError(err)
		s.testUsers[i] = createdUser

		// 確保使用者已經被創建
		_, err = s.userRepo.GetUserByID(createdUser.UserID)
		s.Require().NoError(err)
	}

	// 準備測試商品資料
	s.testProducts = make([]*model.Product, 5) // 建立5個測試商品
	for i := 0; i < 5; i++ {
		productID := fmt.Sprintf("P%d", i+1)
		price := decimal.NewFromFloat(float64((i + 1) * 100))
		s.testProducts[i] = &model.Product{
			ProductID: productID,
			Name:      fmt.Sprintf("Test Product %d", i+1),
			Price:     price,
		}

		initialStock := uint(30 + i) // 每個商品30+i的初始庫存
		// 記錄初始庫存
		s.initialStocks[productID] = initialStock

		// 建立商品和設置庫存
		err := s.productRepo.CreateProductStock(s.ctx, productID, initialStock)
		s.Require().NoError(err)

		// 確保商品和庫存都已經被創建
		stock, err := s.productRepo.GetProductStock(s.ctx, productID)
		s.Require().NoError(err)
		s.Equal(initialStock, stock)
	}

	// 在設置完成後，給予一些時間讓系統穩定
	time.Sleep(2 * time.Second)
	s.T().Log("End SetupTest")
}

// TearDownTest 在每個測試結束後執行
func (s *EventDBQueryTestSuite) TearDownTest() {
	s.T().Log("Start TearDownTest...")

	// 清理測試使用者資料
	for _, user := range s.testUsers {
		err := s.userRepo.HardDeleteUser(user.UserID)
		s.Require().NoError(err)
	}

	// 清理測試商品資料
	for _, product := range s.testProducts {
		// 清理商品庫存（Redis）
		err := s.productRepo.AddProductStock(s.ctx, product.ProductID, 0)
		s.Require().NoError(err)
		// 清理商品資料
		err = s.productRepo.DeleteProductStock(s.ctx, product.ProductID)
		s.Require().NoError(err)
	}

	// 等待一段時間確保所有連接都已正確關閉
	time.Sleep(2 * time.Second)
	s.T().Log("End TearDownTest")
}

// TestProjectionConnection 測試 Projection 連線
func (s *EventDBQueryTestSuite) TestProjectionConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// 準備測試數據
	orderID := testOrderID
	streamID := GenerateOrderStreamID(orderID)
	fmt.Printf("訂單ID: %s\n", orderID)
	fmt.Printf("事件流ID (streamID): %s\n", streamID)
	fmt.Printf("查詢用的 Partition ID: %s\n", orderID)

	userID := 1001
	items := []model.OrderItemData{
		{
			ProductID:   "prod-1",
			ProductName: "測試商品1",
			Quantity:    2,
			Price:       decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(200),
		},
	}
	amount := decimal.NewFromInt(200)

	// 創建訂單事件
	orderCreatedEvt := evt_model.NewOrderCreatedEvent(orderID, userID, orderID, time.Now(), items, amount, 1)

	// 發送事件
	err := s.eventDao.AppendEvent(ctx, orderCreatedEvt.EventID, streamID, string(evt_model.OrderCreatedEventName), orderCreatedEvt)
	s.Require().NoError(err)
	fmt.Printf("事件已發送到流: %s\n", streamID)

	// 等待 projection 處理事件
	time.Sleep(time.Second * 2)

	// 嘗試多次查詢結果
	var aggregate *evt_model.OrderAggregate
	aggregate, err = s.queryDao.GetOrderState(ctx, s.projectionName, streamID)
	if err != nil {
		fmt.Printf("查詢失敗: %v\n", err)
	}
	fmt.Printf("成功查詢到訂單狀態\n")

	// 驗證結果
	s.Require().NoError(err)
	s.NotNil(aggregate)
	s.Equal(orderID, aggregate.OrderID)
	s.Equal(uint(1), aggregate.State)
	s.Len(aggregate.OrderItems, 1)
	s.Equal(items[0].ProductID, aggregate.OrderItems[0].ProductID)

}

// TestOrderEventFlowWithValidation 測試完整訂單流程，每個事件後都進行驗證
func (s *EventDBQueryTestSuite) TestOrderEventFlowWithValidation() {
	// 設定測試數據
	orderID := "test-order-125" // 使用特定的訂單ID
	streamID := GenerateOrderStreamID(orderID)
	userID := 1001
	items := []model.OrderItemData{
		{
			ProductID:   "prod-1",
			ProductName: "測試商品1",
			Quantity:    2,
			Price:       decimal.NewFromInt(100),
			Amount:      decimal.NewFromInt(200),
		},
	}
	amount := decimal.NewFromInt(200)

	// 測試場景1：創建訂單
	s.Run("創建訂單", func() {
		orderCreatedEvt := evt_model.NewOrderCreatedEvent(streamID, userID, orderID, time.Now(), items, amount, 1)

		err := s.eventDao.AppendEvent(s.ctx, orderCreatedEvt.EventID, streamID, string(evt_model.OrderCreatedEventName), orderCreatedEvt)
		s.Require().NoError(err, "發送創建訂單事件失敗")

		// 等待 projection 更新
		time.Sleep(time.Second)

		// 驗證訂單狀態
		aggregate, err := s.queryDao.GetOrderState(s.ctx, s.projectionName, streamID)
		s.Require().NoError(err, "獲取訂單狀態失敗")
		s.Equal(orderID, aggregate.OrderID, "訂單ID不匹配")
		s.Equal(userID, aggregate.UserID, "用戶ID不匹配")
		s.Equal(uint(1), aggregate.State, "訂單狀態不匹配")
		s.Len(aggregate.OrderItems, 1, "訂單項目數量不匹配")
		s.Equal(items[0].ProductID, aggregate.OrderItems[0].ProductID, "商品ID不匹配")
	})

	// 測試場景2：訂單出貨
	s.Run("訂單出貨", func() {
		trackingCode := "SF123456789"
		carrier := "SF Express"
		shippedEvt := evt_model.NewOrderShippedEvent(streamID, trackingCode, carrier, 1, 2)

		err := s.eventDao.AppendEvent(s.ctx, shippedEvt.EventID, streamID, string(evt_model.OrderShippedEventName), shippedEvt)
		s.Require().NoError(err, "發送訂單出貨事件失敗")

		// 等待 projection 更新
		time.Sleep(time.Second)

		// 驗證更新後的狀態
		aggregate, err := s.queryDao.GetOrderState(s.ctx, s.projectionName, streamID)
		s.Require().NoError(err, "獲取訂單狀態失敗")
		s.Equal(uint(2), aggregate.State, "訂單狀態不匹配")
		s.Equal(trackingCode, aggregate.TrackingCode, "追蹤碼不匹配")
		s.Equal(carrier, aggregate.Carrier, "承運商不匹配")
	})

	// 測試場景3：訂單取消
	s.Run("訂單取消", func() {
		cancelMsg := "客戶要求取消"
		cancelEvt := evt_model.NewOrderCancelledEvent(streamID, cancelMsg, 2, 3)

		err := s.eventDao.AppendEvent(s.ctx, cancelEvt.EventID, streamID, string(evt_model.OrderCancelledEventName), cancelEvt)
		s.Require().NoError(err, "發送訂單取消事件失敗")

		// 等待 projection 更新
		time.Sleep(time.Second)

		// 驗證最終狀態
		aggregate, err := s.queryDao.GetOrderState(s.ctx, s.projectionName, streamID)
		s.Require().NoError(err, "獲取訂單狀態失敗")
		s.Equal(uint(3), aggregate.State, "訂單狀態不匹配")
		s.Equal(cancelMsg, aggregate.Message, "取消訊息不匹配")
	})

	// 清理測試數據
	s.cleanupEventStream(s.ctx, streamID)
}

// 運行測試套件
func TestEventDBQuerySuite(t *testing.T) {
	suite.Run(t, new(EventDBQueryTestSuite))
}
