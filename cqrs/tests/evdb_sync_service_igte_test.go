package tests

import (
	"context"
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
		err := s.orderRepo.HardDeleteOrder(orderID)
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
	order, err := s.orderRepo.GetOrderByID(orderID)
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
