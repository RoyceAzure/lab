package tests

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/admin"
	"github.com/stretchr/testify/assert"
)

var testClusterConfig *admin.ClusterConfig

func init() {
	var err error
	testClusterConfig = &admin.ClusterConfig{
		Cluster: struct {
			Name    string   `yaml:"name"`
			Brokers []string `yaml:"brokers"`
		}{
			Name:    "test-cluster",
			Brokers: []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		},
	}
	if err != nil {
		panic(err)
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

// setupTest 為每個測試創建獨立的測試環境
func setupTest(t *testing.T) func() {
	// 創建唯一的 topic 名稱，加入時間戳以便自動清理
	topicName := fmt.Sprintf("test-topic-%s-%d", t.Name(), time.Now().UnixNano())

	// 創建 admin client
	adminClient, err := admin.NewAdmin(testClusterConfig.Cluster.Brokers)
	assert.NoError(t, err)

	// 創建 topic，設定較短的retention時間以便自動清理
	err = adminClient.CreateTopic(context.Background(), admin.TopicConfig{
		Name:              topicName,
		Partitions:        6,
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
	err = adminClient.WaitForTopics(ctx, []string{topicName}, 30*time.Second)
	assert.NoError(t, err)

	// 更新全局配置
	testClusterConfig.Topics = []admin.TopicConfig{
		{
			Name:              topicName,
			Partitions:        6,
			ReplicationFactor: 3,
		},
	}

	// 返回清理函數，只需關閉adminClient
	return func() {
		adminClient.Close()
	}
}
