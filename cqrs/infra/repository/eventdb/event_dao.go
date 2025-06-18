package eventdb

import (
	"context"
	"encoding/json"

	"github.com/EventStore/EventStore-Client-Go/v3/esdb"
)

type EventDao struct {
	client *esdb.Client
}

func NewEventDao(db *esdb.Client) *EventDao {
	return &EventDao{client: db}
}

// 寫入事件（Create）
func (dao *EventDao) AppendEvent(ctx context.Context, streamID, eventType string, data interface{}) error {
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

// 讀取事件（Read）
func (dao *EventDao) ReadEvents(ctx context.Context, streamID string) ([]*esdb.ResolvedEvent, error) {
	opts := esdb.ReadStreamOptions{}
	stream, err := dao.client.ReadStream(ctx, streamID, opts, 100)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var events []*esdb.ResolvedEvent
	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}
		events = append(events, event)
	}
	return events, nil
}

// 刪除事件流（Delete Stream）
func (dao *EventDao) DeleteStream(ctx context.Context, streamID string) error {
	_, err := dao.client.DeleteStream(ctx, streamID, esdb.DeleteStreamOptions{})
	return err
}
