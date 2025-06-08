package test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/admin"
)

var (
	testClusterConfig = &admin.ClusterConfig{
		Cluster: struct {
			Name    string   `yaml:"name"`
			Brokers []string `yaml:"brokers"`
		}{
			Name: "test-cluster",
			Brokers: []string{
				"localhost:9092", // kafka1 映射到本機的端口
				"localhost:9093", // kafka2 映射到本機的端口
				"localhost:9094", // kafka3 映射到本機的端口
			},
		},
		Topics: []admin.TopicConfig{
			{
				Name:              "test-basic-topic",
				Partitions:        3,
				ReplicationFactor: 3,
				Configs: map[string]interface{}{
					"cleanup.policy":       "delete",
					"retention.ms":         "600000",    // 10 分鐘
					"retention.bytes":      "104857600", // 100MB
					"min.insync.replicas":  "2",
					"delete.retention.ms":  "60000",  // 1 分鐘
					"segment.ms":           "300000", // 5 分鐘
					"file.delete.delay.ms": "60000",  // 1 分鐘
				},
			},
			{
				Name:              "test-large-topic",
				Partitions:        6,
				ReplicationFactor: 3,
				Configs: map[string]interface{}{
					"cleanup.policy":       "delete",
					"retention.ms":         "600000",    // 10 分鐘
					"retention.bytes":      "104857600", // 100MB
					"min.insync.replicas":  "2",
					"delete.retention.ms":  "60000",  // 1 分鐘
					"segment.ms":           "300000", // 5 分鐘
					"file.delete.delay.ms": "60000",  // 1 分鐘
				},
			},
		},
	}
)

func setupTestTopics(t *testing.T) func() {
	// 創建 admin client
	adminClient, err := admin.NewAdmin(testClusterConfig.Cluster.Brokers)
	if err != nil {
		t.Fatalf("Failed to create admin client: %v", err)
	}

	// 創建測試需要的 topics
	ctx := context.Background()
	topicNames := make([]string, 0, len(testClusterConfig.Topics))

	for _, topic := range testClusterConfig.Topics {
		topicNames = append(topicNames, topic.Name)
		err := adminClient.CreateTopic(ctx, topic)
		if err != nil {
			// 如果 topic 已存在，我們可以忽略這個錯誤
			if !isTopicExistsError(err) {
				t.Fatalf("Failed to create topic %s: %v", topic.Name, err)
			}
		}
	}

	// 等待 topics 創建完成
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := adminClient.WaitForTopics(waitCtx, topicNames, 20*time.Second); err != nil {
		t.Fatalf("Failed waiting for topics to be ready: %v", err)
	}

	// 驗證 topics 是否可用
	topics, err := adminClient.ListTopics(ctx)
	if err != nil {
		t.Fatalf("Failed to list topics: %v", err)
	}

	// 檢查所有需要的 topics 是否存在
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}

	for _, requiredTopic := range topicNames {
		if !topicMap[requiredTopic] {
			t.Fatalf("Required topic %s was not created", requiredTopic)
		}
	}

	t.Logf("Successfully created and verified topics: %v", topicNames)

	// 返回清理函數
	return func() {
		// 在這裡可以添加清理邏輯，例如刪除測試數據
		adminClient.Close()

		// 等待資源釋放
		time.Sleep(5 * time.Second)
	}
}

func TestMain(m *testing.M) {
	// 在所有測試開始前執行設置
	log.Println("Setting up test environment...")

	// 運行所有測試
	result := m.Run()

	// 測試完成後的清理工作
	log.Println("Cleaning up test environment...")
	time.Sleep(5 * time.Second) // 等待所有資源釋放

	os.Exit(result)
}

// isTopicExistsError 檢查錯誤是否為 topic 已存在的錯誤
func isTopicExistsError(err error) bool {
	return err != nil && err.Error() == "topic already exists"
}

// 每個測試文件都應該在測試開始時調用這個函數
func setupTest(t *testing.T) func() {
	cleanup := setupTestTopics(t)

	// 返回清理函數
	return func() {
		cleanup()
	}
}
