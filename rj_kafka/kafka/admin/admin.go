package admin

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"gopkg.in/yaml.v3"
)

// ClusterConfig 代表整個集群配置
type ClusterConfig struct {
	Cluster struct {
		Name    string   `yaml:"name"`
		Brokers []string `yaml:"brokers"`
	} `yaml:"cluster"`
	Topics  []TopicConfig          `yaml:"topics"`
	ACLs    []ACLConfig            `yaml:"acls"`
	Configs map[string]interface{} `yaml:"configs"`
}

// TopicConfig 代表主題配置
type TopicConfig struct {
	Name              string                 `yaml:"name"`
	Partitions        int                    `yaml:"partitions"`
	ReplicationFactor int                    `yaml:"replication_factor"`
	Configs           map[string]interface{} `yaml:"configs"`
}

// ACLConfig 代表 ACL 配置，用於權限控管
type ACLConfig struct {
	ResourceType   string `yaml:"resource_type"`
	ResourceName   string `yaml:"resource_name"`
	Principal      string `yaml:"principal"`
	Host           string `yaml:"host"`
	Operation      string `yaml:"operation"`
	PermissionType string `yaml:"permission_type"`
}

// TopicConfigUpdate 代表要更新的 topic 配置
type TopicConfigUpdate struct {
	Name    string
	Configs map[string]string
}

// Admin 代表 Kafka 管理工具
type Admin struct {
	conn    *kafka.Conn
	brokers []string
}

// NewAdmin 創建新的 Admin 實例
func NewAdmin(brokers []string) (*Admin, error) {
	var lastErr error
	// 嘗試連接每個 broker 直到找到 controller
	for _, broker := range brokers {
		conn, err := kafka.Dial("tcp", broker)
		if err != nil {
			lastErr = err
			continue
		}

		// 檢查是否為 controller
		controller, err := conn.Controller()
		if err != nil {
			conn.Close()
			lastErr = err
			continue
		}

		// 如果當前 broker 不是 controller，關閉連接並連到 controller
		if controller.Host != broker {
			conn.Close()
			conn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
			if err != nil {
				lastErr = err
				continue
			}
		}

		return &Admin{
			conn:    conn,
			brokers: brokers,
		}, nil
	}

	return nil, fmt.Errorf("failed to connect to any broker and find controller: %v", lastErr)
}

// Close 關閉管理工具
func (a *Admin) Close() error {
	return a.conn.Close()
}

// LoadConfig 從 YAML 檔案載入配置
func LoadConfig(path string) (*ClusterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config ClusterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

// CreateTopic 創建主題
func (a *Admin) CreateTopic(ctx context.Context, topic TopicConfig) error {
	configEntries := make([]kafka.ConfigEntry, 0)
	for key, value := range topic.Configs {
		configEntries = append(configEntries, kafka.ConfigEntry{
			ConfigName:  key,
			ConfigValue: fmt.Sprintf("%v", value),
		})
	}

	err := a.conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic.Name,
		NumPartitions:     topic.Partitions,
		ReplicationFactor: topic.ReplicationFactor,
		ConfigEntries:     configEntries,
	})

	if err != nil {
		return fmt.Errorf("failed to create topic %s: %v", topic.Name, err)
	}

	return nil
}

// DeleteTopic 刪除主題
func (a *Admin) DeleteTopic(ctx context.Context, topicName string) error {
	err := a.conn.DeleteTopics(topicName)
	if err != nil {
		return fmt.Errorf("failed to delete topic %s: %v", topicName, err)
	}
	return nil
}

// ListTopics 列出所有主題
func (a *Admin) ListTopics(ctx context.Context) ([]string, error) {
	partitions, err := a.conn.ReadPartitions()
	if err != nil {
		return nil, fmt.Errorf("failed to read partitions: %v", err)
	}

	topics := make(map[string]struct{})
	for _, p := range partitions {
		topics[p.Topic] = struct{}{}
	}

	result := make([]string, 0, len(topics))
	for topic := range topics {
		result = append(result, topic)
	}

	return result, nil
}

// ApplyConfig 應用配置到集群
func (a *Admin) ApplyConfig(ctx context.Context, config *ClusterConfig) error {
	// 創建主題
	for _, topic := range config.Topics {
		exists, err := a.topicExists(topic.Name)
		if err != nil {
			return err
		}

		if !exists {
			if err := a.CreateTopic(ctx, topic); err != nil {
				return err
			}
		}
	}

	return nil
}

// topicExists 檢查主題是否存在
func (a *Admin) topicExists(topicName string) (bool, error) {
	topics, err := a.ListTopics(context.Background())
	if err != nil {
		return false, err
	}

	for _, topic := range topics {
		if topic == topicName {
			return true, nil
		}
	}

	return false, nil
}

// WaitForTopics 等待主題創建完成
func (a *Admin) WaitForTopics(ctx context.Context, topics []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		existingTopics, err := a.ListTopics(ctx)
		if err != nil {
			return err
		}

		allFound := true
		for _, topic := range topics {
			found := false
			for _, existingTopic := range existingTopics {
				if existingTopic == topic {
					found = true
					break
				}
			}
			if !found {
				allFound = false
				break
			}
		}

		if allFound {
			return nil
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("timeout waiting for topics to be created")
}

// UpdateTopicConfig 更新 topic 的配置
func (a *Admin) UpdateTopicConfig(ctx context.Context, update TopicConfigUpdate) error {
	// 檢查 topic 是否存在
	exists, err := a.topicExists(update.Name)
	if err != nil {
		return fmt.Errorf("failed to check topic existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("topic %s does not exist", update.Name)
	}

	// 準備配置更新
	entries := make([]kafka.ConfigEntry, 0, len(update.Configs))
	for key, value := range update.Configs {
		entries = append(entries, kafka.ConfigEntry{
			ConfigName:  key,
			ConfigValue: value,
		})
	}

	// 使用 CreateTopics 來更新配置
	// 注意：這裡我們只更新配置，不改變分區數
	err = a.conn.CreateTopics(kafka.TopicConfig{
		Topic:         update.Name,
		ConfigEntries: entries,
	})

	if err != nil {
		return fmt.Errorf("failed to update topic config: %v", err)
	}

	return nil
}

// GetTopicConfig 獲取 topic 的當前配置
func (a *Admin) GetTopicConfig(ctx context.Context, topicName string) (map[string]string, error) {
	// 檢查 topic 是否存在
	exists, err := a.topicExists(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %v", err)
	}
	if !exists {
		return nil, fmt.Errorf("topic %s does not exist", topicName)
	}

	// 使用 conn.ReadPartitions 獲取分區資訊
	partitions, err := a.conn.ReadPartitions(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic metadata: %v", err)
	}

	// 獲取第一個分區的配置作為 topic 配置
	if len(partitions) > 0 {
		configs := make(map[string]string)
		// 添加基本配置
		configs["partitions"] = strconv.Itoa(len(partitions))
		configs["replication_factor"] = strconv.Itoa(len(partitions[0].Replicas))

		return configs, nil
	}

	return make(map[string]string), nil
}

// PurgeCluster 完全清除 Kafka 集群的所有資料
// 這個操作會刪除所有 topics，請謹慎使用
func (a *Admin) PurgeCluster(ctx context.Context) error {
	// 1. 列出所有 topics
	topics, err := a.ListTopics(ctx)
	if err != nil {
		return fmt.Errorf("failed to list topics: %v", err)
	}

	// 2. 刪除所有 topics
	for _, topic := range topics {
		// 跳過內部 topics
		if strings.HasPrefix(topic, "__") {
			continue
		}
		if err := a.DeleteTopic(ctx, topic); err != nil {
			return fmt.Errorf("failed to delete topic %s: %v", topic, err)
		}
	}

	// 3. 等待 topics 完全刪除
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		remainingTopics, err := a.ListTopics(ctx)
		if err != nil {
			return fmt.Errorf("failed to check remaining topics: %v", err)
		}

		// 檢查是否還有非內部 topics
		hasUserTopics := false
		for _, topic := range remainingTopics {
			if !strings.HasPrefix(topic, "__") {
				hasUserTopics = true
				break
			}
		}

		if !hasUserTopics {
			return nil
		}

		time.Sleep(time.Second)
	}

	return fmt.Errorf("timeout waiting for topics to be deleted")
}
