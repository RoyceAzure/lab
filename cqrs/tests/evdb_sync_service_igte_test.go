package tests

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EventDBSyncServiceTestSuite struct {
	suite.Suite
	*require.Assertions

	ctx                  context.Context
	orderEventDBQueryDao *eventdb.EventDBQueryDao
	orderRepo            *db.OrderRepo
	syncService          *service.OrderSyncService
	dbConn               *db.DbDao
	esClient             *esdb.Client
	projectionClient     *esdb.ProjectionClient
	eventDBDao           *eventdb.EventDao

	// 用於清理測試數據
	createdOrderIDs []string
}

const (
	projectionName = "test-order-batch-projection"
)

func TestEventDBSyncService(t *testing.T) {
	suite.Run(t, new(EventDBSyncServiceTestSuite))
}

func (s *EventDBSyncServiceTestSuite) SetupSuite() {
	s.Assertions = require.New(s.T())
	s.ctx = context.Background()

	// 初始化資料庫連接
	conn, err := db.GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	s.Require().NoError(err)
	s.dbConn = db.NewDbDao(conn)

	// 初始化資料庫
	err = s.dbConn.InitMigrate()
	s.Require().NoError(err)

	// 初始化 OrderRepo
	s.orderRepo = db.NewOrderRepo(s.dbConn)

	// 設置 EventStore 客戶端
	settings, err := esdb.ParseConnectionString("esdb://localhost:2113?tls=false&keepAliveTimeout=10000")
	s.Require().NoError(err)

	s.esClient, err = esdb.NewClient(settings)
	s.Require().NoError(err)

	s.projectionClient, err = esdb.NewProjectionClient(settings)
	s.Require().NoError(err)

	// 初始化 EventDBQueryDao
	s.orderEventDBQueryDao = eventdb.NewEventDBQueryDao(s.projectionClient, s.esClient)

	// 初始化同步服務
	s.syncService = service.NewEventDBSyncService(s.orderEventDBQueryDao, s.orderRepo)

	// 初始化 EventDao
	s.eventDBDao = eventdb.NewEventDao(s.esClient)

	s.orderEventDBQueryDao.CreateOrderBatchProjection(s.ctx, projectionName)

	// 初始化清理列表
	s.createdOrderIDs = make([]string, 0)
}

func (s *EventDBSyncServiceTestSuite) TearDownSuite() {
	// 關閉資料庫連接
	if s.projectionClient != nil {
		s.projectionClient.Close()
	}
	if s.esClient != nil {
		s.esClient.Close()
	}

	// 清理測試期間創建的訂單
}

func (s *EventDBSyncServiceTestSuite) SetupTest() {
	s.createdOrderIDs = make([]string, 0)
}

func (s *EventDBSyncServiceTestSuite) TearDownTest() {
	// 清理測試期間創建的訂單
	for _, orderID := range s.createdOrderIDs {
		err := s.orderRepo.HardDeleteOrder(s.ctx, orderID)
		if err != nil {
			s.T().Logf("清理訂單 %s 時發生錯誤: %v", orderID, err)
		}
	}
	s.createdOrderIDs = make([]string, 0)
}

func (s *EventDBSyncServiceTestSuite) TestOrderCreateEventSync() {
	// 準備測試數據
	orderID := "666"
	userID := 1
	now := time.Now().UTC()

	// 創建訂單事件
	orderEvent := &evt_model.OrderCreatedEvent{
		BaseEvent: evt_model.BaseEvent{
			EventID:     uuid.New(),
			EventType:   evt_model.OrderCreatedEventName,
			AggregateID: orderID,
			CreatedAt:   now,
		},
		UserID:    userID,
		OrderID:   orderID,
		OrderDate: now,
		Items: []model.OrderItemData{
			{
				OrderID:   orderID,
				ProductID: "prod-1",
				Quantity:  2,
				Price:     decimal.NewFromFloat(100.00),
			},
		},
		Amount:  decimal.NewFromFloat(200.00),
		ToState: uint(evt_model.OrderStateCreated),
	}

	// 啟動同步服務
	err := s.syncService.Start()
	s.Require().NoError(err, "啟動同步服務失敗")
	defer func() {
		err := s.syncService.Stop(5 * time.Second)
		s.Require().NoError(err, "停止同步服務失敗")
	}()

	// 發送訂單創建事件
	err = s.eventDBDao.SaveOrderCreatedEvent(s.ctx, orderEvent)
	s.Require().NoError(err, "發送訂單創建事件失敗")

	// 記錄已創建的訂單ID，用於清理
	s.createdOrderIDs = append(s.createdOrderIDs, orderID)

	// 等待同步處理完成
	time.Sleep(10 * time.Second)

	// 從資料庫查詢訂單
	order, err := s.orderRepo.GetOrderByID(s.ctx, orderID)
	s.Require().NoError(err, "查詢訂單失敗")
	s.Require().NotNil(order, "訂單不應為空")

	// 驗證訂單內容
	s.Equal(orderID, order.OrderID, "訂單ID不匹配")
	s.Equal(userID, order.UserID, "用戶ID不匹配")
	s.Equal(orderEvent.Amount.String(), order.Amount.String(), "訂單總金額不匹配")

	// 驗證訂單項目
	s.Require().Len(order.OrderItems, len(orderEvent.Items), "訂單項目數量不匹配")
	s.Equal(orderEvent.Items[0].ProductID, order.OrderItems[0].ProductID, "商品ID不匹配")
	s.Equal(orderEvent.Items[0].Quantity, order.OrderItems[0].Quantity, "商品數量不匹配")
}

func (s *EventDBSyncServiceTestSuite) TestConcurrentOrderEvents() {
	// 測試配置
	const (
		numUsers     = 5               // 模擬的用戶數量
		testDuration = 5 * time.Second // 測試持續時間
		minOrderID   = 100             // 最小訂單ID
		maxOrderID   = 999             // 最大訂單ID
	)

	// 用於追蹤每個訂單的最後狀態
	type lastEventRecord struct {
		eventType evt_model.EventType
		event     interface{}
	}
	lastEvents := sync.Map{}

	// 生成隨機的訂單ID
	orderIDs := make([]string, numUsers)
	for i := 0; i < numUsers; i++ {
		orderIDs[i] = fmt.Sprintf("%d", rand.Intn(maxOrderID-minOrderID+1)+minOrderID)
		s.createdOrderIDs = append(s.createdOrderIDs, orderIDs[i]) // 用於清理
	}

	// 啟動同步服務
	err := s.syncService.Start()
	s.Require().NoError(err, "啟動同步服務失敗")
	defer func() {
		err := s.syncService.Stop(5 * time.Second)
		s.Require().NoError(err, "停止同步服務失敗")
	}()

	// 啟動多個goroutine模擬用戶操作
	var wg sync.WaitGroup
	startTime := time.Now()

	for _, orderID := range orderIDs {
		wg.Add(1)
		go func(orderID string) {
			defer wg.Done()

			for time.Since(startTime) < testDuration {
				// 隨機選擇一個事件類型
				event := s.generateRandomOrderEvent(orderID)

				// 儲存事件
				var err error
				switch e := event.(type) {
				case *evt_model.OrderCreatedEvent:
					err = s.eventDBDao.SaveOrderCreatedEvent(s.ctx, e)
				case *evt_model.OrderConfirmedEvent:
					err = s.eventDBDao.SaveOrderConfirmedEvent(s.ctx, e)
				case *evt_model.OrderShippedEvent:
					err = s.eventDBDao.SaveOrderShippedEvent(s.ctx, e)
				case *evt_model.OrderCancelledEvent:
					err = s.eventDBDao.SaveOrderCancelledEvent(s.ctx, e)
				case *evt_model.OrderRefundedEvent:
					err = s.eventDBDao.SaveOrderRefundedEvent(s.ctx, e)
				}

				s.Require().NoError(err, "儲存事件失敗")

				// 記錄最後的事件
				if time.Since(startTime) >= testDuration {
					lastEvents.Store(orderID, lastEventRecord{
						eventType: s.getEventType(event),
						event:     event,
					})
					break
				}

				// 隨機休眠一段時間
				time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			}
		}(orderID)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 等待同步處理完成
	time.Sleep(10 * time.Second)

	// 驗證每個訂單的最後狀態
	lastEvents.Range(func(key, value interface{}) bool {
		orderID := key.(string)
		record := value.(lastEventRecord)

		// 從 PostgreSQL 資料庫查詢訂單
		order, err := s.orderRepo.GetOrderByID(s.ctx, orderID)
		s.Require().NoError(err, "查詢訂單失敗")
		s.Require().NotNil(order, "訂單不應為空")

		// 根據最後事件類型驗證訂單狀態
		switch record.eventType {
		case evt_model.OrderCreatedEventName:
			evt := record.event.(*evt_model.OrderCreatedEvent)
			s.Equal(evt.UserID, order.UserID, "用戶ID不匹配")
			s.Equal(evt.Amount.String(), order.Amount.String(), "訂單金額不匹配")
			s.Equal(uint(model.OrderStatusPending), order.State, "訂單狀態不匹配")
		case evt_model.OrderConfirmedEventName:
			s.Equal(uint(model.OrderStatusConfirmed), order.State, "訂單狀態不匹配")
		case evt_model.OrderShippedEventName:
			s.Equal(uint(model.OrderStatusShipped), order.State, "訂單狀態不匹配")
		case evt_model.OrderCancelledEventName:
			s.Equal(uint(model.OrderStatusCancelled), order.State, "訂單狀態不匹配")
		case evt_model.OrderRefundedEventName:
			s.Equal(uint(model.OrderStatusRefunded), order.State, "訂單狀態不匹配")
		}

		return true
	})
}

// generateRandomOrderEvent 生成隨機的訂單事件
func (s *EventDBSyncServiceTestSuite) generateRandomOrderEvent(orderID string) interface{} {
	now := time.Now()

	// 隨機選擇一個事件類型
	eventTypes := []func() interface{}{
		func() interface{} {
			return &evt_model.OrderCreatedEvent{
				BaseEvent: evt_model.BaseEvent{
					EventID:     uuid.New(),
					EventType:   evt_model.OrderCreatedEventName,
					AggregateID: orderID,
					CreatedAt:   now,
				},
				UserID:    rand.Intn(1000), // 隨機用戶ID
				OrderID:   orderID,
				OrderDate: now,
				Items: []model.OrderItemData{
					{
						OrderID:   orderID,
						ProductID: fmt.Sprintf("prod_%d", rand.Intn(100)),
						Quantity:  rand.Intn(5) + 1,
						Price:     decimal.NewFromFloat(float64(rand.Intn(100)) + 0.99),
					},
				},
				Amount:  decimal.NewFromFloat(float64(rand.Intn(1000)) + 0.99),
				ToState: uint(model.OrderStatusPending),
			}
		},
		func() interface{} {
			return &evt_model.OrderConfirmedEvent{
				BaseEvent: evt_model.BaseEvent{
					EventID:     uuid.New(),
					EventType:   evt_model.OrderConfirmedEventName,
					AggregateID: orderID,
					CreatedAt:   now,
				},
				FromState: uint(model.OrderStatusPending),
				ToState:   uint(model.OrderStatusConfirmed),
			}
		},
		func() interface{} {
			return &evt_model.OrderShippedEvent{
				BaseEvent: evt_model.BaseEvent{
					EventID:     uuid.New(),
					EventType:   evt_model.OrderShippedEventName,
					AggregateID: orderID,
					CreatedAt:   now,
				},
				TrackingCode: fmt.Sprintf("TRK%d", rand.Int63()),
				Carrier:      "TestCarrier",
				FromState:    uint(model.OrderStatusConfirmed),
				ToState:      uint(model.OrderStatusShipped),
			}
		},
		func() interface{} {
			return &evt_model.OrderCancelledEvent{
				BaseEvent: evt_model.BaseEvent{
					EventID:     uuid.New(),
					EventType:   evt_model.OrderCancelledEventName,
					AggregateID: orderID,
					CreatedAt:   now,
				},
				Message:   "Test cancellation",
				FromState: uint(model.OrderStatusPending),
				ToState:   uint(model.OrderStatusCancelled),
			}
		},
		func() interface{} {
			return &evt_model.OrderRefundedEvent{
				BaseEvent: evt_model.BaseEvent{
					EventID:     uuid.New(),
					EventType:   evt_model.OrderRefundedEventName,
					AggregateID: orderID,
					CreatedAt:   now,
				},
				Amount:    decimal.NewFromFloat(float64(rand.Intn(1000)) + 0.99),
				FromState: uint(model.OrderStatusCancelled),
				ToState:   uint(model.OrderStatusRefunded),
			}
		},
	}

	return eventTypes[rand.Intn(len(eventTypes))]()
}

// getEventType 獲取事件的類型
func (s *EventDBSyncServiceTestSuite) getEventType(event interface{}) evt_model.EventType {
	switch e := event.(type) {
	case *evt_model.OrderCreatedEvent:
		return e.EventType
	case *evt_model.OrderConfirmedEvent:
		return e.EventType
	case *evt_model.OrderShippedEvent:
		return e.EventType
	case *evt_model.OrderCancelledEvent:
		return e.EventType
	case *evt_model.OrderRefundedEvent:
		return e.EventType
	default:
		return ""
	}
}
