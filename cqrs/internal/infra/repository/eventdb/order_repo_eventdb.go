package eventdb

import (
	"context"
	"encoding/json"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
)

//

// 處理order相關事件儲存到eventdb
// order聚合的還原  projection
// 後續應該統一給app層 order service 處理
func (dao *EventDao) test(ctx context.Context, streamID, eventType string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	eventData := esdb.EventData{
		ContentType: esdb.ContentTypeJson,
		EventType:   eventType,
		Data:        payload,
	}
	_, err = dao.client.AppendToStream(ctx, streamID, esdb.AppendToStreamOptions{}, eventData)
	return err
}
