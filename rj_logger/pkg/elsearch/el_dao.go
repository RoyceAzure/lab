package elsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/olivere/elastic/v7"
)

var (
	instance *ElSearchDao
	mu       sync.Mutex
)

type IElSearchDao interface {
	Create(index string, data []byte) error
	Read(index, id string) ([]byte, error)
	Update(index, id string, data []byte) error
	Delete(index, id string) error
	Search(index string, query elastic.Query) ([]byte, error)
	Close() error
}

type ElSearchDao struct {
	client *elastic.Client
}

// url :connect string example : "http://localhost:9200"
func InitELSearch(url, pas string, options ...elastic.ClientOptionFunc) error {
	if instance != nil {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return nil
	}

	dao, err := newElSearchDao(url, pas, options...)
	if err != nil {
		return err
	}

	instance = dao

	return nil
}

func GetInstance() (*ElSearchDao, error) {
	if instance == nil {
		return nil, fmt.Errorf("please call InitELSearch first")
	}

	return instance, nil
}

func newElSearchDao(url, pas string, options ...elastic.ClientOptionFunc) (*ElSearchDao, error) {
	defaultOptions := []elastic.ClientOptionFunc{
		elastic.SetURL(url),                  // ES 服務器地址
		elastic.SetSniff(false),              // 禁用 Sniffing
		elastic.SetHealthcheck(true),         // 啟用健康檢查
		elastic.SetBasicAuth("elastic", pas), // 基本認證
		elastic.SetRetrier(elastic.NewBackoffRetrier(elastic.NewExponentialBackoff(time.Second, 30*time.Second))), // 重試策略
		elastic.SetGzip(true), // 啟用 Gzip 壓縮
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)), // 錯誤日誌
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)),          // 訊息日誌
	}

	options = append(defaultOptions, options...)

	client, err := elastic.NewClient(options...)
	if err != nil {
		return nil, err
	}

	return &ElSearchDao{
		client: client,
	}, err
}

type BaseLog struct {
	ID          string    `json:"id,omitempty"`
	Message     string    `json:"message"`
	Level       string    `json:"level"`
	Timestamp   time.Time `json:"timestamp"`
	ProjectName string    `json:"project_name"`
	ModuleName  string    `json:"module_name"`
	ActionName  string    `json:"action_name"`
}

// 泛型日誌結構
type Log struct {
	BaseLog
	Data any `json:"data"`
}

// Create 泛型創建方法
func (e *ElSearchDao) Create(index string, data []byte) error {
	var anyJson map[string]any

	err := json.Unmarshal(data, &anyJson)
	if err != nil {
		return err
	}

	_, err = e.client.Index().
		Index(index).
		BodyJson(anyJson).
		Do(context.Background())
	return err
}

// Read 單筆讀取
func (e *ElSearchDao) Read(index, id string) ([]byte, error) {
	result, err := e.client.Get().
		Index(index).
		Id(id).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	return []byte(result.Source), nil
}

// Update 更新
func (e *ElSearchDao) Update(index, id string, data []byte) error {
	var anyJson interface{}
	if err := json.Unmarshal(data, &anyJson); err != nil {
		return err
	}

	_, err := e.client.Update().
		Index(index).
		Id(id).
		Doc(anyJson).
		Do(context.Background())
	return err
}

// Delete 刪除
func (e *ElSearchDao) Delete(index, id string) error {
	_, err := e.client.Delete().
		Index(index).
		Id(id).
		Do(context.Background())
	return err
}

// Search 範圍查詢
func (e *ElSearchDao) Search(index string, query elastic.Query) ([]byte, error) {
	result, err := e.client.Search().
		Index(index).
		Query(query).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	// 把 hits 中的數據轉換成我們需要的格式
	hits := make([]map[string]interface{}, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		// 解析 Source
		var sourceMap map[string]interface{}
		if err := json.Unmarshal(hit.Source, &sourceMap); err != nil {
			return nil, err
		}
		// 添加 _id
		sourceMap["id"] = hit.Id
		hits[i] = sourceMap
	}

	// 將整個結果轉換為 JSON
	return json.Marshal(hits)
}
func (e *ElSearchDao) Close() error {
	e.client.Stop()
	return nil
}
