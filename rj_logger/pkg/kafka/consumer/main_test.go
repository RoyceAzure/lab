package consumer

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_logger/internal/logger_consumer"
	"github.com/RoyceAzure/lab/rj_logger/pkg/elsearch"
	"github.com/segmentio/kafka-go"
)

const (
	elasticHost                = "http://localhost:9200"
	elasticPassword            = "password"
	defaultConsummerBufferSize = 1000
	testTopic                  = "test-log-topic"
	groupID                    = "test-consumer-group"
)

var (
	reader     *kafka.Reader
	writer     *kafka.Writer
	brokers    = []string{"localhost:9092", "localhost:9093", "localhost:9094"}
	consumer   *BaseConsumer
	processer  Processer
	elasticDao *elsearch.ElSearchDao
)

func TestMain(m *testing.M) {
	// setup
	setUpReader()
	setUpElasticDao()
	setUpKafkaElProcesser()
	setUpKafkaConsumer()
	code := m.Run() // 執行所有測試

	// teardown
	closeTest()
	os.Exit(code)
}

func setUpReader() {
	// 配置 reader
	reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   testTopic,
		GroupID: groupID,
		// 從第一條消息開始讀取
		StartOffset: kafka.FirstOffset,
		// 設置較長的超時時間
		MaxWait: 3 * time.Second,
	})
}

func setUpElasticDao() {
	err := elsearch.InitELSearch(elasticHost, elasticPassword)
	if err != nil {
		panic(err)
	}

	elasticDao, err = elsearch.GetInstance()
	if err != nil {
		panic(err)
	}
}

func setUpKafkaElProcesser() {
	// 配置 reader
	if elasticDao == nil {
		panic(fmt.Errorf("reader is not init"))
	}
	var err error
	processer, err = logger_consumer.NewKafkaElProcesser(elasticDao, defaultConsummerBufferSize)
	if err != nil {
		panic(err)
	}
}

func setUpKafkaConsumer() {
	if reader == nil {
		panic(fmt.Errorf("reader is not init"))
	}
	if processer == nil {
		panic(fmt.Errorf("processer is not init"))
	}
	consumer = NewBaseConsumer(reader, processer)
	consumer.Start()
}

func closeTest() {
	var errs []error

	if err := consumer.Stop(5 * time.Second); err != nil {
		errs = append(errs, err)
	}
	if err := reader.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := elasticDao.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		log.Printf("cleanup errors: %v", errs)
	}
}
