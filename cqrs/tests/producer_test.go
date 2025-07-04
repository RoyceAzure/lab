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
	cmd_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/command"
	command_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/command"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/handler/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/consumer"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/producer"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/producer/balancer"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/admin"
	kafka_config "github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	kafka_consumer "github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	kafka_producer "github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	testClusterConfig *admin.ClusterConfig

	ConsumerGroupPrefix     = fmt.Sprintf("test-cqrs-group")
	CartCmdConsumerGroup    = fmt.Sprintf("%s-cart-cmd", ConsumerGroupPrefix)
	CartEventConsumerGroup  = fmt.Sprintf("%s-cart-evt", ConsumerGroupPrefix)
	OrderEventConsumerGroup = fmt.Sprintf("%s-order-evt", ConsumerGroupPrefix)

	TopicPrefix         = fmt.Sprintf("test-cqrs-topic")
	CartCmdTopicName    = fmt.Sprintf("%s-cart-cmd-%d", TopicPrefix, time.Now().UnixNano())
	CartEventTopicName  = fmt.Sprintf("%s-cart-evt-%d", TopicPrefix, time.Now().UnixNano())
	OrderEventTopicName = fmt.Sprintf("%s-order-evt-%d", TopicPrefix, time.Now().UnixNano())

	//projection 固定用一個處理order category就可以
	OrderProjectionName = "test-order-projection"

	ProductNum = 10
	UserNum    = 100
)

var kafkaConfigTemplate kafka_config.Config

// setupTest 為每個測試創建獨立的測試環境
func setupTestEnvironment(t *testing.T) func() {
	testClusterConfig = &admin.ClusterConfig{
		Cluster: struct {
			Name    string   `yaml:"name"`
			Brokers []string `yaml:"brokers"`
		}{
			Name:    "test-cluster",
			Brokers: []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		},
	}

	kafkaConfigTemplate = kafka_config.Config{
		Brokers:        testClusterConfig.Cluster.Brokers,
		Partition:      6,
		RetryAttempts:  3,
		BatchTimeout:   time.Second,
		BatchSize:      1,
		RequiredAcks:   1,
		CommitInterval: time.Second,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		Balancer:       balancer.NewCartBalancer(6),
	}

	// 創建 admin client
	adminClient, err := admin.NewAdmin(testClusterConfig.Cluster.Brokers)
	assert.NoError(t, err)

	// 創建 cart command topic，設定較短的retention時間以便自動清理
	err = adminClient.CreateTopic(context.Background(), admin.TopicConfig{
		Name:              CartCmdTopicName,
		Partitions:        kafkaConfigTemplate.Partition,
		ReplicationFactor: 3,
		Configs: map[string]interface{}{
			"cleanup.policy":      "delete",
			"retention.ms":        "60000", // 1分鐘後自動刪除
			"min.insync.replicas": "2",
			"delete.retention.ms": "1000", // 標記刪除後1秒鐘清理
		},
	})
	assert.NoError(t, err)

	// 創建 cart event topic，設定較短的retention時間以便自動清理
	err = adminClient.CreateTopic(context.Background(), admin.TopicConfig{
		Name:              CartEventTopicName,
		Partitions:        kafkaConfigTemplate.Partition,
		ReplicationFactor: 3,
		Configs: map[string]interface{}{
			"cleanup.policy":      "delete",
			"retention.ms":        "60000", // 1分鐘後自動刪除
			"min.insync.replicas": "2",
			"delete.retention.ms": "1000", // 標記刪除後1秒鐘清理
		},
	})
	assert.NoError(t, err)

	// 等待 topic 創建完成
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = adminClient.WaitForTopics(ctx, []string{CartCmdTopicName, CartEventTopicName}, 30*time.Second)
	assert.NoError(t, err)

	// 更新全局配置
	testClusterConfig.Topics = []admin.TopicConfig{
		{
			Name:              CartCmdTopicName,
			Partitions:        kafkaConfigTemplate.Partition,
			ReplicationFactor: 3,
		},
		{
			Name:              CartEventTopicName,
			Partitions:        kafkaConfigTemplate.Partition,
			ReplicationFactor: 3,
		},
	}

	// 返回清理函數，只需關閉adminClient
	return func() {
		adminClient.Close()
	}
}

// 購物車相關topic: cart userID 做key
// 6個partitions 6個消費者
// product相關topic: product productID 做key
// 6個partitions 6個消費者
type ProducerTestSuite struct {
	suite.Suite
	dbDao                 *db.DbDao
	userRepo              *db.UserRepo
	userOrderRepo         *db.UserOrderRepo
	productRepo           *redis_repo.ProductRepo
	orderEventDB          *eventdb.EventDao
	userService           *service.UserService
	productService        *service.ProductService
	orderService          *service.OrderService
	cartRepo              *redis_repo.CartRepo
	orderEventHandler     event_handler.Handler
	cartEventHandler      event_handler.Handler
	cartCommandHandler    command_handler.Handler
	cartCommandProducer   *producer.CartCommandProducer //for測試使用者發送命令
	cartCommandConsumers  []consumer.IBaseConsumer
	cartEventConsumers    []consumer.IBaseConsumer
	toCartCommandProducer kafka_producer.Producer //for測試使用者發送命令
	toCartEventProducer   kafka_producer.Producer //for cart command handler
	orderEventDBQueryDao  *eventdb.EventDBQueryDao

	redisClient *redis.Client
	// 測試資料
	testUsers     []*model.User
	testProducts  []*model.Product
	initialStocks map[string]uint // 記錄每個商品的初始庫存
}

func TestProducerTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}

func generateRandomProductID() string {
	return fmt.Sprintf("P%d", rand.Intn(1000000))
}

func (suite *ProducerTestSuite) SetupSuite() {
	suite.T().Log("=== SetupSuite: 開始設置測試套件 ===")
	cleanup := setupTestEnvironment(suite.T())
	defer cleanup()

	ctx := context.Background()
	// 初始化 DB 連線
	conn, err := db.GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(suite.T(), err)
	suite.dbDao = db.NewDbDao(conn)

	err = suite.dbDao.InitMigrate()
	require.NoError(suite.T(), err)

	// 初始化 Redis 連線
	suite.redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password",
		DB:       1, // 使用測試用的DB
	})

	// 清空測試用的 Redis DB
	err = suite.redisClient.FlushDB(ctx).Err()
	require.NoError(suite.T(), err)

	// 初始化 repositories
	suite.setupOrderEventDBQueryDao()

	suite.userRepo = db.NewUserRepo(suite.dbDao)
	suite.productRepo = redis_repo.NewProductRepo(suite.redisClient)
	suite.userOrderRepo = db.NewUserOrderRepo(suite.dbDao)
	suite.cartRepo = redis_repo.NewCartRepo(suite.redisClient)
	suite.userService = service.NewUserService(suite.userRepo)
	suite.productService = service.NewProductService(suite.productRepo)
	suite.setupToCartCommandProducer()
	suite.setupCartCommandProducer()
	suite.setupCartEventConsumer()
	suite.setupCartCommandConsumer()

	suite.T().Log("=== SetupSuite: 測試套件設置完成 ===")
}

func (suite *ProducerTestSuite) SetupTest() {
	suite.T().Log("=== SetupTest: 開始設置單個測試 ===")

	ctx := context.Background()

	// 初始化 initialStocks map
	suite.initialStocks = make(map[string]uint)

	// 準備測試使用者資料
	suite.testUsers = make([]*model.User, UserNum)
	for i := 0; i < UserNum; i++ {
		user := &model.User{
			UserName:    fmt.Sprintf("Test User %d", i+1),
			UserEmail:   fmt.Sprintf("test%d@example.com", i+1),
			UserPhone:   fmt.Sprintf("09%d", i+1),
			UserAddress: "test address",
		}
		// 建立測試使用者並獲取創建後的用戶（包含自動生成的ID）
		createdUser, err := suite.userRepo.CreateUser(user)
		require.NoError(suite.T(), err)
		suite.testUsers[i] = createdUser

		// 確保使用者已經被創建
		_, err = suite.userRepo.GetUserByID(createdUser.UserID)
		require.NoError(suite.T(), err)
	}

	// 準備測試商品資料
	suite.testProducts = make([]*model.Product, ProductNum)
	for i := 0; i < ProductNum; i++ {
		productID := generateRandomProductID()
		price := decimal.NewFromFloat(float64((i + 1) * 100))
		suite.testProducts[i] = &model.Product{
			ProductID: productID,
			Name:      fmt.Sprintf("Test Product %d", i+1),
			Price:     price,
		}

		initialStock := uint(rand.Intn(100) + 30)
		// 記錄初始庫存
		suite.initialStocks[productID] = initialStock

		// 建立商品
		err := suite.productRepo.CreateProductStock(ctx, productID, initialStock)
		require.NoError(suite.T(), err)

		// 設置商品庫存
		err = suite.productRepo.CreateProductStock(ctx, productID, initialStock)
		require.NoError(suite.T(), err)

		// 確保商品和庫存都已經被創建
		_, err = suite.productRepo.GetProductStock(ctx, productID)
		require.NoError(suite.T(), err)
		stock, err := suite.productRepo.GetProductStock(ctx, productID)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), initialStock, stock)
	}

	// 在設置完成後，給予一些時間讓系統穩定
	time.Sleep(2 * time.Second)
	suite.T().Logf("=== SetupTest: 測試設置完成，已創建 %d 個使用者和 %d 個商品 ===", len(suite.testUsers), len(suite.testProducts))
}

func (suite *ProducerTestSuite) TearDownTest() {
	suite.T().Log("=== TearDownTest: 開始清理單個測試 ===")
	ctx := context.Background()

	// 清理測試使用者資料
	for _, user := range suite.testUsers {
		err := suite.userRepo.HardDeleteUser(user.UserID)
		require.NoError(suite.T(), err)
	}

	// 清理測試商品資料
	for _, product := range suite.testProducts {
		// 清理商品庫存（Redis）
		err := suite.productRepo.AddProductStock(ctx, product.ProductID, 0)
		require.NoError(suite.T(), err)
		// 清理商品資料（DB）
		err = suite.productRepo.DeleteProductStock(ctx, product.ProductID)
		require.NoError(suite.T(), err)
	}

	// 清理user order
	for _, user := range suite.testUsers {
		suite.userOrderRepo.HardDeleteUserOrder(user.UserID)
	}

	// 等待一段時間確保所有連接都已正確關閉
	time.Sleep(2 * time.Second)

	suite.T().Log("=== TearDownTest: 測試清理完成 ===")
}

func (suite *ProducerTestSuite) TearDownSuite() {
	suite.T().Log("=== TearDownSuite: 開始清理測試套件 ===")
	// 先停止所有消費者
	wg := sync.WaitGroup{}
	for _, cus := range suite.cartCommandConsumers {
		wg.Add(1)
		go func(consumer consumer.IBaseConsumer) {
			defer wg.Done()
			consumer.Stop()
		}(cus)
	}
	for _, cus := range suite.cartEventConsumers {
		wg.Add(1)
		go func(consumer consumer.IBaseConsumer) {
			defer wg.Done()
			consumer.Stop()
		}(cus)
	}

	wg.Wait()

	// 關閉生產者
	if suite.toCartCommandProducer != nil {
		suite.toCartCommandProducer.Close()
		suite.toCartCommandProducer = nil
	}
	if suite.toCartEventProducer != nil {
		suite.toCartEventProducer.Close()
		suite.toCartEventProducer = nil
	}
	suite.T().Log("=== TearDownSuite: 測試套件清理完成 ===")
}

// for user
func (suite *ProducerTestSuite) setupToCartCommandProducer() {
	configTemplate := kafkaConfigTemplate
	configTemplate.Topic = CartCmdTopicName
	p, err := kafka_producer.New(&configTemplate)
	require.NoError(suite.T(), err)
	suite.toCartCommandProducer = p
}

// for cart command handler
func (suite *ProducerTestSuite) setupToForCartEventProducer() {
	configTemplate := kafkaConfigTemplate
	configTemplate.Topic = CartEventTopicName
	p, err := kafka_producer.New(&configTemplate)
	require.NoError(suite.T(), err)
	suite.toCartEventProducer = p
}

// 設置producer
// 設置NewCartCommandProducer
func (suite *ProducerTestSuite) setupCartCommandProducer() {
	if suite.toCartCommandProducer == nil {
		suite.setupToCartCommandProducer()
	}
	if suite.toCartCommandProducer == nil {
		suite.T().Fatalf("kafkaProducer for cart command handler is not initialized")
	}

	suite.cartCommandProducer = producer.NewCartCommandProducer(suite.toCartCommandProducer)
}

// set up order event db query dao
func (suite *ProducerTestSuite) setupOrderEventDBQueryDao() {
	// 設置 EventStore 客戶端
	settings, err := esdb.ParseConnectionString("esdb://localhost:2113?tls=false&keepAliveTimeout=10000")
	require.NoError(suite.T(), err)

	client, err := esdb.NewClient(settings)
	require.NoError(suite.T(), err)

	projectionClient, err := esdb.NewProjectionClient(settings)
	require.NoError(suite.T(), err)

	// 初始化 orderEventDB
	suite.orderEventDB = eventdb.NewEventDao(client)

	suite.orderEventDBQueryDao = eventdb.NewEventDBQueryDao(projectionClient, client)

	suite.orderEventDBQueryDao.CreateOrderProjection(context.Background(), OrderProjectionName)
}

func (suite *ProducerTestSuite) setupCartCommandHandler() {
	if suite.toCartEventProducer == nil {
		suite.setupToForCartEventProducer()
	}
	if suite.toCartEventProducer == nil {
		suite.T().Fatalf("kafkaProducer for cart command handler is not initialized")
	}
	suite.cartCommandHandler = command_handler.NewCartCommandHandler(
		suite.userService,
		suite.productService,
		suite.cartRepo,
		suite.userOrderRepo,
		suite.orderEventDB,
		suite.toCartEventProducer,
	)
}

// 設置cart command handler
// 建立n個cart command consumer
func (suite *ProducerTestSuite) setupCartCommandConsumer() {
	if suite.cartCommandHandler == nil {
		suite.setupCartCommandHandler()
	}
	if suite.cartCommandHandler == nil {
		suite.T().Fatalf("cartCommandHandler is not initialized")
	}

	configTemplate := kafkaConfigTemplate
	configTemplate.ConsumerGroup = CartCmdConsumerGroup
	configTemplate.Topic = CartCmdTopicName
	partitions := configTemplate.Partition
	for i := 0; i < partitions; i++ {
		cos, err := kafka_consumer.New(&configTemplate)
		require.NoError(suite.T(), err)
		cartCommandConsumer := consumer.NewCartCommandConsumer(cos, suite.cartCommandHandler)
		cartCommandConsumer.Start(context.Background())
		suite.cartCommandConsumers = append(suite.cartCommandConsumers, cartCommandConsumer)
	}
}

func (suite *ProducerTestSuite) setupCartEventHandler() {
	suite.cartEventHandler = event_handler.NewCartEventHandler(suite.cartRepo)
}

// 設置cart event handler
// 建立n個cart event consumer
// consumer要不一樣的group  for cart event
func (suite *ProducerTestSuite) setupCartEventConsumer() {
	if suite.cartEventHandler == nil {
		suite.setupCartEventHandler()
	}
	if suite.cartEventHandler == nil {
		suite.T().Fatalf("cartEventHandler is not initialized")
	}

	configTemplate := kafkaConfigTemplate
	configTemplate.ConsumerGroup = CartEventConsumerGroup
	configTemplate.Topic = CartEventTopicName
	partitions := configTemplate.Partition
	for i := 0; i < partitions; i++ {
		cos, err := kafka_consumer.New(&configTemplate)
		require.NoError(suite.T(), err)
		cartEventConsumer := consumer.NewCartEventConsumer(cos, suite.cartEventHandler)
		cartEventConsumer.Start(context.Background())
		suite.cartEventConsumers = append(suite.cartEventConsumers, cartEventConsumer)
	}
}

// simulateUserCartOperations 模擬單一使用者的購物車操作
// 回傳 error channel，可用於監控操作過程中的錯誤
func (suite *ProducerTestSuite) simulateUserCartOperations(ctx context.Context, userID int, duration time.Duration, errChan chan error) {
	// 記錄使用者對每個商品的操作數量
	userProductQuantities := make(map[string]int)

	// 第一步：建立購物車，隨機選擇1-3個商品
	initialItemCount := rand.Intn(3) + 1
	initialItems := make([]model.CartItem, 0, initialItemCount)

	// 避免重複選擇同一個商品
	selectedProducts := make(map[string]struct{})
	for i := 0; i < initialItemCount; i++ {
		// 隨機選擇一個未被選過的商品
		var product *model.Product
		for {
			product = suite.testProducts[rand.Intn(len(suite.testProducts))]
			if _, exists := selectedProducts[product.ProductID]; !exists {
				selectedProducts[product.ProductID] = struct{}{}
				break
			}
		}

		quantity := rand.Intn(5) + 1 // 1-5個商品
		initialItems = append(initialItems, model.CartItem{
			ProductID: product.ProductID,
			Quantity:  quantity,
		})
		// 記錄初始購物車的商品數量
		userProductQuantities[product.ProductID] = quantity
	}

	// 發送創建購物車命令
	if err := suite.cartCommandProducer.ProduceCartCreatedCommand(ctx, userID, initialItems); err != nil {
		errChan <- fmt.Errorf("failed to create cart for user %d: %w", userID, err)
		return
	}

	// 建立計時器
	timer := time.NewTimer(duration)
	defer timer.Stop()

	// 持續發送更新指令直到時間到
	for {
		select {
		case <-ctx.Done():
			// suite.printUserProductQuantities(userID, userProductQuantities)
			return
		case <-timer.C:
			// suite.printUserProductQuantities(userID, userProductQuantities)
			// 發送確認命令
			if err := suite.cartCommandProducer.ProduceCartConfirmedCommand(ctx, userID); err != nil {
				errChan <- fmt.Errorf("failed to confirm cart for user %d: %w", userID, err)
			}
			return
		default:
			// 隨機等待 100-300ms，避免請求太密集
			time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)

			// 隨機決定要更新幾個商品（1-3個）
			updateCount := rand.Intn(3) + 1
			details := make([]cmd_model.CartUpdatedDetial, 0, updateCount)

			// 隨機選擇商品進行更新
			for i := 0; i < updateCount; i++ {
				product := suite.testProducts[rand.Intn(len(suite.testProducts))]
				action := cmd_model.CartAddItem
				quantity := rand.Intn(3) + 1 // 1-3的數量變化

				details = append(details, cmd_model.CartUpdatedDetial{
					Action:    action,
					ProductID: product.ProductID,
					Quantity:  quantity,
				})

				// 記錄商品數量變化，明確處理 key 不存在的情況
				currentQuantity, exists := userProductQuantities[product.ProductID]
				if !exists {
					userProductQuantities[product.ProductID] = quantity
				} else {
					userProductQuantities[product.ProductID] = currentQuantity + quantity
				}
			}

			// 發送更新命令
			if err := suite.cartCommandProducer.ProduceCartUpdatedCommand(ctx, userID, details); err != nil {
				errChan <- fmt.Errorf("failed to update cart for user %d: %w", userID, err)
				continue
			}
		}
	}
}

// 新增一個輔助函數來打印使用者的商品數量
func (suite *ProducerTestSuite) printUserProductQuantities(userID int, quantities map[string]int) {
	suite.T().Logf("=== User %d Product Quantities ===", userID)
	for productID, quantity := range quantities {
		suite.T().Logf("Product %s: %d items", productID, quantity)
	}
	suite.T().Logf("================================")
}

func (suite *ProducerTestSuite) TestConcurrentCartOperations() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 收集所有使用者的錯誤通道
	errChan := make(chan error, 500)

	go func() {
		for range errChan {
		}
	}()

	// 啟動所有使用者的操作
	wg := sync.WaitGroup{}
	suite.T().Logf("start to simulate user cart operations")
	for _, user := range suite.testUsers {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			suite.simulateUserCartOperations(ctx, userID, 20*time.Second, errChan)
		}(user.UserID)
	}

	wg.Wait()
	suite.T().Logf("all users cart operations completed")
	close(errChan)

	suite.T().Logf("all users cart operations completed")

	//驗證階段
	// 等待一段時間讓事件，projection被處理
	time.Sleep(40 * time.Second)

	// 1. 獲取所有商品的當前庫存
	currentStocks := make(map[string]uint)
	for _, product := range suite.testProducts {
		stock, err := suite.productRepo.GetProductStock(ctx, product.ProductID)
		require.NoError(suite.T(), err)
		currentStocks[product.ProductID] = stock
	}

	// 2. 獲取所有使用者的購物車內容，並驗證訂單
	cartQuantities := make(map[string]uint)
	for _, user := range suite.testUsers {
		// 檢查購物車內容
		cart, err := suite.cartRepo.Get(ctx, user.UserID)
		if err != nil {
			continue // 跳過空購物車
		}

		// 檢查訂單內容
		userOrders, err := suite.userOrderRepo.ListUserOrdersByUserID(user.UserID)
		require.NoError(suite.T(), err)
		//測試情況下，使用者只會有一筆訂單
		require.Equal(suite.T(), 1, len(userOrders))

		// 取得訂單詳細內容
		orderID := userOrders[0].OrderID
		order, err := suite.orderEventDBQueryDao.GetOrderState(ctx, OrderProjectionName, orderID)
		require.NoError(suite.T(), err)

		// 驗證訂單基本資訊
		require.Equal(suite.T(), user.UserID, order.UserID, "Order UserID mismatch")

		// 驗證訂單內容與購物車內容一致
		require.Equal(suite.T(), len(cart.OrderItems), len(order.OrderItems),
			"Order items count should match cart items count for user %d", user.UserID)

		// 建立訂單項目的 map 以便比較
		orderItemMap := make(map[string]int)
		for _, item := range order.OrderItems {
			orderItemMap[item.ProductID] = item.Quantity
		}
		// 比較購物車和訂單中的每個商品
		for _, cartItem := range cart.OrderItems {
			orderQuantity, exists := orderItemMap[cartItem.ProductID]
			require.True(suite.T(), exists,
				"Product %s in cart should exist in order for user %d",
				cartItem.ProductID, user.UserID)
			require.Equal(suite.T(), cartItem.Quantity, orderQuantity,
				"Product %s quantity mismatch between cart and order for user %d",
				cartItem.ProductID, user.UserID)
		}

		// 累計商品總數量
		for _, item := range cart.OrderItems {
			cartQuantities[item.ProductID] += uint(item.Quantity)
		}
	}

	// 3. 驗證每個商品的庫存變化是否與購物車數量相符
	for productID, initialStock := range suite.initialStocks {
		currentStock := currentStocks[productID]
		inCarts := cartQuantities[productID]

		// 初始庫存 - 當前庫存 = 購物車中的總數量
		suite.T().Logf("Product %s: initial=%d, current=%d, in_carts=%d",
			productID, initialStock, currentStock, inCarts)

		require.Equal(suite.T(), initialStock-currentStock, inCarts,
			"Product %s: stock difference should equal tota/l in carts", productID)
	}
}
