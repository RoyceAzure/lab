package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/admin"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/config"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/consumer"
	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
)

func main() {
	// 1. 載入叢集配置
	clusterCfg, err := admin.LoadConfig("./deployment/kafka/cluster-config.yaml")
	if err != nil {
		log.Fatalf("Failed to load cluster config: %v", err)
	}

	// 2. 初始化 Admin 並應用配置
	adminClient, err := admin.NewAdmin(clusterCfg.Cluster.Brokers)
	if err != nil {
		log.Fatalf("Failed to create admin client: %v", err)
	}
	defer adminClient.Close()

	ctx := context.Background()
	if err := adminClient.ApplyConfig(ctx, clusterCfg); err != nil {
		log.Fatalf("Failed to apply cluster config: %v", err)
	}

	// 3. 建立 Producer 配置
	producerCfg := &config.Config{
		Brokers:       clusterCfg.Cluster.Brokers,
		Topic:         "test-topic", // 使用配置文件中定義的主題
		RetryAttempts: 3,
		BatchTimeout:  time.Second,
		BatchSize:     100,
		RequiredAcks:  -1, // 等待所有副本確認
	}

	// 4. 建立 Producer
	p, err := producer.New(producerCfg)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer p.Close()

	// 5. 建立 Consumer 配置
	consumerCfg := &config.Config{
		Brokers:          clusterCfg.Cluster.Brokers,
		Topic:            "test-topic",
		ConsumerGroup:    "test-group",
		ConsumerMinBytes: 10e3, // 10KB
		ConsumerMaxBytes: 10e6, // 10MB
		ConsumerMaxWait:  time.Second,
		CommitInterval:   time.Second,
	}

	// 6. 建立 Consumer
	c, err := consumer.New(consumerCfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer c.Close()

	// 7. 設置優雅關閉
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 處理系統信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 8. 啟動 Consumer
	msgChan, errChan, err := c.Consume()
	if err != nil {
		log.Fatalf("Failed to consume messages: %v", err)
	}

	// 9. 主要處理邏輯
	go func() {
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				log.Printf("Received message: %s\n", string(msg.Value))
				// 處理訊息...

			case err, ok := <-errChan:
				if !ok {
					return
				}
				log.Printf("Error: %v\n", err)
				// 處理錯誤...

			case <-ctx.Done():
				return
			}
		}
	}()

	// 10. 等待關閉信號
	<-sigChan
	log.Println("Shutting down...")
	cancel()
	c.Close()

	// 給一些時間讓現有的操作完成
	time.Sleep(time.Second * 2)
}
