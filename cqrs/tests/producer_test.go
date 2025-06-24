package tests

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	command_handler "github.com/RoyceAzure/lab/cqrs/internal/command/handler"
	"github.com/RoyceAzure/lab/cqrs/internal/consumer"
	event_handler "github.com/RoyceAzure/lab/cqrs/internal/event/handler"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db/model"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/RoyceAzure/lab/cqrs/internal/producer"
	"github.com/RoyceAzure/lab/cqrs/internal/service"
	kafka_config "github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	kafka_consumer "github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	kafka_producer "github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var kafkaConfigTemplate = kafka_config.Config{
	Brokers:        testClusterConfig.Cluster.Brokers,
	Topic:          testClusterConfig.Topics[0].Name, // 使用動態創建的 topic
	ConsumerGroup:  fmt.Sprintf("test-group-command-producer-consumer%d", time.Now().UnixNano()),
	RetryAttempts:  3,
	BatchTimeout:   time.Second,
	BatchSize:      1,
	RequiredAcks:   1,
	CommitInterval: time.Second,
	ReadTimeout:    5 * time.Second,
	WriteTimeout:   5 * time.Second,
}

// 購物車相關topic: cart userID 做key
// 6個partitions 6個消費者
// product相關topic: product productID 做key
// 6個partitions 6個消費者
type ProducerTestSuite struct {
	suite.Suite
	dbDao                *db.DbDao
	userRepo             *db.UserRepo
	productRepo          *redis_repo.ProductRepo
	userService          *service.UserService
	productService       *service.ProductService
	cartRepo             *redis_repo.CartRepo
	cartEventHandler     event_handler.Handler
	cartCommandHandler   command_handler.Handler
	cartCommandProducer  *producer.CartCommandProducer
	cartCommandConsumers []consumer.IBaseConsumer
	cartEventConsumers   []consumer.IBaseConsumer
	kafkaProducer        kafka_producer.Producer //共用，要給cart command producer, cart command handler 使用
	kafkaConsumers       []kafka_consumer.Consumer

	redisClient *redis.Client
	// 測試資料
	testUsers    []*model.User
	testProducts []*model.Product
}

func TestProducerTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerTestSuite))
}

func generanteRandomint() int {
	return rand.Intn(1000000)
}

func generateRandomProductID() string {
	return fmt.Sprintf("P%d", rand.Intn(1000000))
}

func (suite *ProducerTestSuite) SetupSuite() {
	ctx := context.Background()
	// 初始化 DB 連線
	conn, err := db.GetDbConn("lab_cqrs", "localhost", "5432", "royce", "password")
	require.NoError(suite.T(), err)
	suite.dbDao = db.NewDbDao(conn)

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
	suite.userRepo = db.NewUserRepo(suite.dbDao)
	suite.productRepo = redis_repo.NewProductRepo(suite.redisClient)

}

func (suite *ProducerTestSuite) SetupTest() {
	ctx := context.Background()

	// 準備測試使用者資料
	userNum := 10
	suite.testUsers = make([]*model.User, userNum)
	for i := 0; i < userNum; i++ {
		suite.testUsers[i] = &model.User{
			UserID:    generanteRandomint(),
			UserName:  fmt.Sprintf("Test User %d", i+1),
			UserEmail: fmt.Sprintf("test%d@example.com", i+1),
		}
	}

	// 建立測試使用者
	for _, user := range suite.testUsers {
		err := suite.userRepo.CreateUser(user)
		require.NoError(suite.T(), err)
		// 確保使用者已經被創建
		_, err = suite.userRepo.GetUserByID(user.UserID)
		require.NoError(suite.T(), err)
	}

	// 準備測試商品資料
	productNum := 20
	suite.testProducts = make([]*model.Product, productNum)
	for i := 0; i < productNum; i++ {
		productID := generateRandomProductID()
		price := decimal.NewFromFloat(float64((i + 1) * 100))
		suite.testProducts[i] = &model.Product{
			ProductID: productID,
			Name:      fmt.Sprintf("Test Product %d", i+1),
			Price:     price,
		}

		// 建立商品
		err := suite.productRepo.CreateProductStock(ctx, productID, uint((i+1)*10))
		require.NoError(suite.T(), err)

		// 設置商品庫存
		err = suite.productRepo.CreateProductStock(ctx, productID, uint((i+1)*10))
		require.NoError(suite.T(), err)

		// 確保商品和庫存都已經被創建
		_, err = suite.productRepo.GetProductStock(ctx, productID)
		require.NoError(suite.T(), err)
		stock, err := suite.productRepo.GetProductStock(ctx, productID)
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), uint((i+1)*10), stock)
	}
}

func (suite *ProducerTestSuite) TearDownTest() {
	ctx := context.Background()

	// 清理測試使用者資料
	for _, user := range suite.testUsers {
		err := suite.userRepo.HardDeleteUser(user.UserID)
		require.NoError(suite.T(), err)
	}

	// 清理測試商品資料
	for _, product := range suite.testProducts {
		// 清理商品庫存（Redis）
		err := suite.productRepo.DeltaProductStock(ctx, product.ProductID, 0)
		require.NoError(suite.T(), err)
		// 清理商品資料（DB）
		err = suite.productRepo.DeleteProductStock(ctx, product.ProductID)
		require.NoError(suite.T(), err)
	}
}

func (suite *ProducerTestSuite) TearDownSuite() {
}

func (suite *ProducerTestSuite) setupProducer() {
	p, err := kafka_producer.New(&kafkaConfigTemplate)
	require.NoError(suite.T(), err)
	suite.kafkaProducer = p
}

func (suite *ProducerTestSuite) setupCartCommandProducer() {
	if suite.kafkaProducer == nil {
		suite.setupProducer()
	}
	if suite.kafkaProducer == nil {
		suite.T().Fatalf("kafkaProducer is not initialized")
	}

	suite.cartCommandProducer = producer.NewCartCommandProducer(suite.kafkaProducer)
}

func (suite *ProducerTestSuite) setupCartCommandHandler() {
	suite.cartCommandHandler = command_handler.NewCartCommandHandler(suite.userService, suite.productService, suite.kafkaProducer)
}

// 建立6個cart command消費者
func (suite *ProducerTestSuite) setupCartCommandConsumer() {
	if suite.cartCommandHandler == nil {
		suite.setupCartCommandHandler()
	}
	if suite.cartCommandHandler == nil {
		suite.T().Fatalf("cartCommandHandler is not initialized")
	}

	partitions := testClusterConfig.Topics[0].Partitions
	for i := 0; i < partitions; i++ {
		cos, err := kafka_consumer.New(&kafkaConfigTemplate)
		require.NoError(suite.T(), err)
		cartCommandConsumer := consumer.NewCartCommandConsumer(cos, suite.cartCommandHandler)
		cartCommandConsumer.Start(context.Background())
		suite.cartCommandConsumers = append(suite.cartCommandConsumers, cartCommandConsumer)
	}
}

// consumer要不一樣的group  for cart event
func (suite *ProducerTestSuite) setupCartEventConsumer() {
	partitions := testClusterConfig.Topics[0].Partitions
	for i := 0; i < partitions; i++ {
		cos, err := kafka_consumer.New(&kafkaConfigTemplate)
		require.NoError(suite.T(), err)
		cartEventConsumer := consumer.NewCartEventConsumer(cos, suite.cartEventHandler)
		cartEventConsumer.Start(context.Background())
		suite.cartEventConsumers = append(suite.cartEventConsumers, cartEventConsumer)
	}
}
