package intergration_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/admin"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/message"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/RoyceAzure/rj/infra/elsearch"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

var (
	testClusterConfig *admin.ClusterConfig
	cfg               *config.Config
	ConsuemErrorGroup = "consumer-group"
	LogConsumerGropu  = fmt.Sprintf("%s-log-%d", ConsuemErrorGroup, time.Now().UnixNano())

	TopicPrefix  = "test-topic"
	LogTopicName = fmt.Sprintf("%s-log-%d", TopicPrefix, time.Now().UnixNano())
)

type TestMsg struct {
	Id      int    `json:"id"`
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

func generateTestMessage(n int) []message.Message {
	t := make([]message.Message, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(i))
		testMsg := TestMsg{
			Id:      i,
			Message: fmt.Sprintf("this is test message %d", i),
		}

		b, err := json.Marshal(testMsg)
		if err != nil {
			panic(err)
		}
		t = append(t, message.Message{
			Key:   buf,
			Value: b,
		})
	}
	return t
}

func handleResult(allTestMsgs, sendFailedMsgs []message.Message, consumSuccess, consumFailed []kafka.Message) []map[string]struct{} {
	allRes := make([]map[string]struct{}, 2)
	allTestMsgsM := make(map[string]struct{}, len(allTestMsgs))
	for _, v := range allTestMsgs {
		allTestMsgsM[string(v.ToKafkaMessage().Key)] = struct{}{}
	}
	allRes[0] = allTestMsgsM

	allSendMsgs := make(map[string]struct{}, len(allTestMsgs)+len(sendFailedMsgs)+len(consumSuccess)+len(consumFailed))
	for _, v := range sendFailedMsgs {
		allSendMsgs[string(v.ToKafkaMessage().Key)] = struct{}{}
	}

	for _, v := range consumSuccess {
		allSendMsgs[string(v.Key)] = struct{}{}
	}

	for _, v := range consumFailed {
		allSendMsgs[string(v.Key)] = struct{}{}
	}
	allRes[1] = allSendMsgs
	return allRes
}

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

	cfg = config.DefaultConfig()
	cfg.Brokers = testClusterConfig.Cluster.Brokers
	cfg.Topic = LogTopicName
	cfg.Partition = 6
	cfg.Timeout = time.Second
	cfg.BatchSize = 10000
	cfg.RequiredAcks = 0 //writer不需要等待確認
	cfg.CommitInterval = time.Millisecond * 500

	// 創建 admin client
	adminClient, err := admin.NewAdmin(testClusterConfig.Cluster.Brokers)
	assert.NoError(t, err)

	// 創建 cart command topic，設定較短的retention時間以便自動清理
	err = adminClient.CreateTopic(context.Background(), admin.TopicConfig{
		Name:              LogTopicName,
		Partitions:        cfg.Partition,
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
	err = adminClient.WaitForTopics(ctx, []string{LogTopicName}, 30*time.Second)
	assert.NoError(t, err)

	// 更新全局配置
	testClusterConfig.Topics = []admin.TopicConfig{
		{
			Name:              LogTopicName,
			Partitions:        cfg.Partition,
			ReplicationFactor: 3,
		},
	}

	// 返回清理函數，只需關閉adminClient
	return func() {
		adminClient.Close()
	}
}

func setUpWriter() (*kafka.Writer, error) {
	w := kafka.Writer{
		Addr:         kafka.TCP(testClusterConfig.Cluster.Brokers...),
		Topic:        LogTopicName,
		Balancer:     cfg.GetBalancer(),
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.CommitInterval + time.Millisecond*100,
		RequiredAcks: kafka.RequiredAcks(cfg.RequiredAcks),
	}

	return &w, nil
}

func setUpReaders(n int) ([]*kafka.Reader, error) {
	readers := make([]*kafka.Reader, 0, n)
	for i := 0; i < n; i++ {
		readers = append(readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:     testClusterConfig.Cluster.Brokers,
			Topic:       LogTopicName,
			GroupID:     LogConsumerGropu,
			StartOffset: kafka.LastOffset,
		}))
	}
	return readers, nil
}

func setUpConsumers(n int, reader []*kafka.Reader, processer *consumer.KafkaElProcesser, handlerSuccessfunc func(kafka.Message), handlerErrorfunc func(consumer.ConsuemError)) ([]*consumer.Consumer, error) {
	consumers := make([]*consumer.Consumer, 0, n)
	for i := 0; i < n; i++ {
		consumers = append(consumers, consumer.NewConsumer(reader[i], processer, *cfg,
			consumer.SetHandlerSuccessfunc(handlerSuccessfunc),
			consumer.SetHandlerFailedfunc(handlerErrorfunc)))
	}
	return consumers, nil
}

func setUpElDao() (*elsearch.ElSearchDao, error) {
	// 初始化 ES
	err := elsearch.InitELSearch("http://localhost:9200", "password")
	if err != nil {
		return nil, err
	}

	dao, err := elsearch.GetInstance()
	if err != nil {
		return nil, err
	}
	return dao, nil
}

func TestProducerAdbvance(t *testing.T) {

	testCases := []struct {
		name                  string
		each_publish_num      int
		each_publish_duration time.Duration
		testMsgs              int
		generateTestMsg       func(int) []message.Message
		earilyStop            time.Duration
		handlerSuccessfunc    func(successMsgsChan chan kafka.Message, successDeposeCount *atomic.Uint32) func(kafka.Message)
		handlerErrorfunc      func(failedMsgsChan chan kafka.Message, failedDeposeCount *atomic.Uint32) func(consumer.ConsuemError)
	}{
		{
			name:                  "all send, all consume",
			testMsgs:              100000,
			each_publish_num:      100,
			each_publish_duration: 10 * time.Millisecond,
			generateTestMsg:       generateTestMessage,
			earilyStop:            time.Second * 20,
			handlerSuccessfunc: func(successMsgsChan chan kafka.Message, successDeposeCount *atomic.Uint32) func(kafka.Message) {
				return func(msg kafka.Message) {
					select {
					case successMsgsChan <- msg:
					default:
						successDeposeCount.Add(1)
					}
				}
			},
			handlerErrorfunc: func(failedMsgsChan chan kafka.Message, failedDeposeCount *atomic.Uint32) func(consumer.ConsuemError) {
				return func(err consumer.ConsuemError) {
					select {
					case failedMsgsChan <- err.Message:
					default:
						failedDeposeCount.Add(1)
					}
				}
			},
		},
		{
			name:                  "all send, all consume, low publish num and duration",
			testMsgs:              100000,
			each_publish_num:      1,
			each_publish_duration: 10 * time.Nanosecond,
			generateTestMsg:       generateTestMessage,
			earilyStop:            time.Second * 20,
			handlerSuccessfunc: func(successMsgsChan chan kafka.Message, successDeposeCount *atomic.Uint32) func(kafka.Message) {
				return func(msg kafka.Message) {
					select {
					case successMsgsChan <- msg:
					default:
						successDeposeCount.Add(1)
					}
				}
			},
			handlerErrorfunc: func(failedMsgsChan chan kafka.Message, failedDeposeCount *atomic.Uint32) func(consumer.ConsuemError) {
				return func(err consumer.ConsuemError) {
					select {
					case failedMsgsChan <- err.Message:
					default:
						failedDeposeCount.Add(1)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			closeEnv := setupTestEnvironment(t)
			defer closeEnv()

			successMsgsChan, failedMsgsChan := make(chan kafka.Message, 10000), make(chan kafka.Message, 10000)
			successMsgs := make([]kafka.Message, 0, 10000)
			failedMsgs := make([]kafka.Message, 0, 10000)

			var successDeposeCount, failedDeposeCount atomic.Uint32
			successDeposeCount.Store(0)
			failedDeposeCount.Store(0)

			writer, err := setUpWriter()
			require.NoError(t, err)

			reader, err := setUpReaders(cfg.Partition)
			require.NoError(t, err)

			elDao, err := setUpElDao()
			require.NoError(t, err)

			processer, err := consumer.NewKafkaElProcesser(elDao, cfg.CommitInterval)
			require.NoError(t, err)

			concumser, err := setUpConsumers(cfg.Partition,
				reader, processer,
				tc.handlerSuccessfunc(successMsgsChan, &successDeposeCount),
				tc.handlerErrorfunc(failedMsgsChan, &failedDeposeCount),
			)
			require.NoError(t, err)

			producer, err := producer.NewConcurrencekafkaProducer(writer, *cfg)
			require.NoError(t, err)
			for _, consumer := range concumser {
				consumer.Start()
			}

			producer.Start()

			time.Sleep(time.Second * 10) // 等待kafka相關初始化

			testMsgs := tc.generateTestMsg(tc.testMsgs)
			sendFailedMsg := make([]message.Message, 0, tc.testMsgs)

			testEnd := make(chan struct{})

			g := new(errgroup.Group)
			g.Go(func() error {
				for msg := range successMsgsChan {
					successMsgs = append(successMsgs, msg)
				}
				return nil // 或返回錯誤
			})

			g.Go(func() error {
				for msg := range failedMsgsChan {
					failedMsgs = append(failedMsgs, msg)
				}
				return nil // 或返回錯誤
			})

			go func() {
				defer func() {
					t.Log("calling writer Close()")
					err = producer.Close(time.Second * 30)
					testEnd <- struct{}{}
				}()
				buffer := tc.each_publish_num
				start := 0
				end := start + buffer
				l := len(testMsgs)

				ticker := time.NewTicker(tc.each_publish_duration)
				defer ticker.Stop()
				for range ticker.C {
					if end > l {
						end = l
					}
					sendFailed, err := producer.Produce(context.Background(), testMsgs[start:end])
					if err != nil {
						sendFailedMsg = append(sendFailedMsg, sendFailed...)
					}
					if end == l {
						return
					}
					start = end
					end += buffer
				}
			}()

			<-testEnd
			if err != nil {
				t.Logf("error: %s", err.Error())
			}

			time.Sleep(time.Second * 20)
			for _, consumer := range concumser {
				consumer.Close(time.Second * 20)
			}

			for _, consumer := range concumser {
				<-consumer.C()
			}
			close(successMsgsChan)
			close(failedMsgsChan)
			// require.Nil(t, err)

			g.Wait()

			eachRes := handleResult(testMsgs, sendFailedMsg, successMsgs, failedMsgs)

			require.Equal(t, len(eachRes[0]), len(eachRes[1]))
			require.Equal(t, eachRes[0], eachRes[1], "all test res and all writer res are not equal")
		})
	}
}
