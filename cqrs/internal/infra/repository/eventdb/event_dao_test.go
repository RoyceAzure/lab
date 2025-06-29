package eventdb

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/stretchr/testify/require"
)

type TestEvent struct {
	OrderID   uint
	UserID    uint
	Amount    string
	OrderDate time.Time
}

func TestEventDao(t *testing.T) {
	// 設定 EventStoreDB 連線參數
	settings, err := esdb.ParseConnectionString("esdb://localhost:2113?tls=false")
	require.NoError(t, err, "連線字串解析失敗")

	// 建立連線
	db, err := esdb.NewClient(settings)
	require.NoError(t, err, "建立 EventStore 連線失敗")
	defer db.Close()

	dao := NewEventDao(db)
	version, err := dao.client.GetServerVersion()
	require.NoError(t, err, "取得伺服器版本失敗")
	require.NotEmpty(t, version, "伺服器版本應該不為空")

	// 新增事件測試
	streamID := "order-1001"
	event := TestEvent{
		OrderID:   1001,
		UserID:    1,
		Amount:    "999.99",
		OrderDate: time.Now(),
	}
	err = dao.AppendEvent(context.Background(), streamID, "OrderCreated", event)
	require.NoError(t, err, "寫入事件失敗")

	// 讀取事件測試
	events, err := dao.ReadEvents(context.Background(), streamID)
	require.NoError(t, err, "讀取事件失敗")
	require.NotEmpty(t, events, "事件流應該有事件")

	// 驗證事件內容
	var got TestEvent
	found := false
	for _, evt := range events {
		if evt.OriginalEvent().EventType == "OrderCreated" {
			err := json.Unmarshal(evt.OriginalEvent().Data, &got)
			require.NoError(t, err, "事件資料解析失敗")
			require.Equal(t, event.OrderID, got.OrderID)
			require.Equal(t, event.UserID, got.UserID)
			require.Equal(t, event.Amount, got.Amount)
			found = true
		}
	}
	require.True(t, found, "應該能找到 OrderCreated 事件")

	// 清理測試資料
	err = dao.DeleteStream(context.Background(), streamID)
	require.NoError(t, err, "刪除測試事件流失敗")
}
