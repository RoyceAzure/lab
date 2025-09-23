package elsearch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
)

func TestElSearchDao(t *testing.T) {
	// 初始化 ES
	err := InitELSearch("http://localhost:9200", "password")
	if err != nil {
		t.Fatal(err)
	}

	dao, err := GetInstance()
	if err != nil {
		t.Fatal(err)
	}

	// 測試數據
	type UserAction struct {
		UserID string `json:"user_id"`
		Action string `json:"action"`
	}

	log := Log{
		BaseLog: BaseLog{
			Message:     "test message",
			Level:       "info",
			Timestamp:   time.Now(),
			ProjectName: "test-project",
			ModuleName:  "test-module",
			ActionName:  "test-action",
		},
		Data: UserAction{
			UserID: "123",
			Action: "login",
		},
	}

	// 測試 Create
	t.Run("Test Create", func(t *testing.T) {
		// 檢查 log 內容
		t.Logf("Original log: %+v", log)

		blog, err := json.Marshal(log)
		if err != nil {
			t.Fatal(err)
		}

		// 檢查 JSON
		t.Logf("JSON data: %s", string(blog))

		err = dao.Create("test-index", blog)
		if err != nil {
			t.Fatal(err)
		}
	})

	// 等待 ES 索引
	time.Sleep(1 * time.Second)

	// 測試 Search
	t.Run("Test Search", func(t *testing.T) {
		query := elastic.NewMatchQuery("message", "test message")
		bytelogs, err := dao.Search("test-index", query)
		if err != nil {
			t.Fatal(err)
		}
		if len(bytelogs) == 0 {
			t.Fatal("no logs found")
		}

		modelLogs := []Log{}

		err = json.Unmarshal(bytelogs, &modelLogs)
		if err != nil {
			t.Fatal(err)
		}
	})

	// 測試 Update
	t.Run("Test Update", func(t *testing.T) {
		query := elastic.NewMatchQuery("message", "test message")
		bytes, err := dao.Search("test-index", query)
		if err != nil {
			t.Fatal(err)
		}
		if len(bytes) == 0 {
			t.Fatal("no logs found")
		}

		modelLogs := []Log{}

		err = json.Unmarshal(bytes, &modelLogs)
		if err != nil {
			t.Fatal(err)
		}

		toUpdate := modelLogs[0]

		toUpdate.BaseLog.Message = "updated message"
		blog, err := json.Marshal(toUpdate)
		if err != nil {
			t.Fatal(err)
		}
		err = dao.Update("test-index", toUpdate.ID, blog)
		if err != nil {
			t.Fatal(err)
		}
	})

	// 測試 Delete
	t.Run("Test Delete", func(t *testing.T) {
		query := elastic.NewMatchQuery("message", "test message")
		bytes, err := dao.Search("test-index", query)
		if err != nil {
			t.Fatal(err)
		}
		if len(bytes) == 0 {
			t.Fatal("no logs found")
		}

		modelLogs := []Log{}

		err = json.Unmarshal(bytes, &modelLogs)
		if err != nil {
			t.Fatal(err)
		}

		toDelete := modelLogs[0]

		err = dao.Delete("test-index", toDelete.ID)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Test BatchInsert", func(t *testing.T) {
		// 準備測試數據
		documents := []map[string]interface{}{
			{
				"title":     "Test Doc 1",
				"content":   "This is test document 1",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			{
				"title":     "Test Doc 2",
				"content":   "This is test document 2",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			{
				"title":     "Test Doc 3",
				"content":   "This is test document 3",
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}

		err := dao.BatchInsert("test-index", documents)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Successfully inserted %d documents", len(documents))
	})
}
