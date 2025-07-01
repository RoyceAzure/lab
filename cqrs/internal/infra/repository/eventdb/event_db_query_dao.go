package eventdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/EventStore/EventStore-Client-Go/v4/esdb"
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/shopspring/decimal"
)

type CategoryName string
type ProjectionName string

const (
	OrderCategoryName   CategoryName   = "order"
	OrderProjectionName ProjectionName = "order-projection"
)

// EventDBQueryDao 用於查詢 EventStore 的 DAO
type EventDBQueryDao struct {
	ProjectionClient *esdb.ProjectionClient
	Client           *esdb.Client
}

// NewEventDBQueryDao 創建新的 EventDBQueryDao
func NewEventDBQueryDao(projectionClient *esdb.ProjectionClient, client *esdb.Client) *EventDBQueryDao {
	return &EventDBQueryDao{
		ProjectionClient: projectionClient,
		Client:           client,
	}
}

// projection
// CreateProjection 創建一個新的 projection
func (e *EventDBQueryDao) CreateOrderProjection(ctx context.Context, projectionName string) error {
	// 定義 JavaScript projection
	projectionJS := `
	fromCategory('order')
	.partitionBy(function(event) {
        // 使用 streamId 作為分區鍵
        return event.streamId;
    })
	.when({
		$init: function() {
            return {
                order_id: null,
                user_id: null,
                items: [],
                amount: "0",
                order_date: null,
                state: null,
                created_at: null,
                tracking_code: null,
                carrier: null,
                message: null
            };
        },
		'OrderCreated': function(s, e) {
			var data = e.data;
			s.order_id = data.order_id;
			s.user_id = data.user_id;
			s.order_date = data.order_date;
			s.items = data.items;
			s.amount = data.amount;
			s.state = data.to_state;
			s.created_at = data.order_date;
		},
		'OrderConfirmed': function(s, e) {
			var data = e.data;
			s.state = data.to_state;
		},
		'OrderShipped': function(s, e) {
			var data = e.data;
			s.state = data.to_state;
			s.tracking_code = data.tracking_code;
			s.carrier = data.carrier;
		},
		'OrderCancelled': function(s, e) {
			var data = e.data;
			s.state = data.to_state;
			s.message = data.message;
		},
		'OrderRefunded': function(s, e) {
			var data = e.data;
			s.state = data.to_state;
			s.amount = data.amount;
		}
	})
	.outputState();
	`
	err := e.ProjectionClient.Create(ctx, projectionName, projectionJS, esdb.CreateProjectionOptions{})
	if err != nil {
		if isProjectionConflictError(err) {
			return nil
		}
		return fmt.Errorf("failed to create projection: %w", err)
	}

	return nil
}

// 使用projectionName和streamId分區查詢
// projections-{projectName}-{streamId}-result
// param:
//
//	projectionName: 投影名稱
//	orderId: 訂單id
//
// return:
//
//	訂單狀態
//	error: 錯誤
func (e *EventDBQueryDao) GetOrderState(ctx context.Context, projectionName string, orderId string) (*evt_model.OrderAggregate, error) {
	// 構建結果流名稱
	streamID := GenerateOrderStreamID(orderId)
	resultStreamID := fmt.Sprintf("$projections-%s-%s-result", projectionName, streamID)

	// 讀取結果流
	stream, err := e.Client.ReadStream(ctx, resultStreamID, esdb.ReadStreamOptions{
		Direction: esdb.Backwards,
		From:      esdb.End{},
	}, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to read stream %s: %w", streamID, err)
	}
	defer stream.Close()

	// 讀取最新的事件
	resolvedEvent, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive event: %w", err)
	}

	if resolvedEvent == nil || len(resolvedEvent.Event.Data) == 0 {
		return nil, fmt.Errorf("no state found for order %s", streamID)
	}

	// 將事件資料轉換為 map
	var resultMap map[string]interface{}
	if err := json.Unmarshal(resolvedEvent.Event.Data, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	// 創建 OrderAggregate
	aggregate := &evt_model.OrderAggregate{}

	// 安全地提取和轉換值
	if v, ok := resultMap["order_id"]; ok {
		if s, ok := v.(string); ok {
			aggregate.OrderID = s
		}
	}

	if v, ok := resultMap["user_id"]; ok {
		if f, ok := v.(float64); ok {
			aggregate.UserID = int(f)
		}
	}

	if v, ok := resultMap["amount"]; ok {
		if f, ok := v.(float64); ok {
			aggregate.Amount = decimal.NewFromFloat(f)
		}
	}

	if v, ok := resultMap["state"]; ok {
		if f, ok := v.(float64); ok {
			aggregate.State = uint(f)
		}
	}

	if v, ok := resultMap["items"]; ok {
		if items, ok := v.([]interface{}); ok {
			orderItems := make([]model.OrderItemData, 0, len(items))
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					orderItem := model.OrderItemData{}

					if pid, ok := itemMap["product_id"].(string); ok {
						orderItem.ProductID = pid
					}
					if qty, ok := itemMap["quantity"].(float64); ok {
						orderItem.Quantity = int(qty)
					}
					if price, ok := itemMap["price"].(float64); ok {
						orderItem.Price = decimal.NewFromFloat(price)
					}
					if amount, ok := itemMap["amount"].(float64); ok {
						orderItem.Amount = decimal.NewFromFloat(amount)
					}
					if name, ok := itemMap["product_name"].(string); ok {
						orderItem.ProductName = name
					}

					orderItems = append(orderItems, orderItem)
				}
			}
			aggregate.OrderItems = orderItems
		}
	}

	if v, ok := resultMap["created_at"]; ok {
		if ts, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				aggregate.OrderDate = t
			}
		}
	}

	if v, ok := resultMap["tracking_code"]; ok {
		if s, ok := v.(string); ok {
			aggregate.TrackingCode = s
		}
	}

	if v, ok := resultMap["carrier"]; ok {
		if s, ok := v.(string); ok {
			aggregate.Carrier = s
		}
	}

	if v, ok := resultMap["message"]; ok {
		if s, ok := v.(string); ok {
			aggregate.Message = s
		}
	}

	return aggregate, nil
}

// DeleteOrderProjection 刪除訂單的 projection
func (e *EventDBQueryDao) DeleteOrderProjection(ctx context.Context, projectionName string) error {
	err := e.ProjectionClient.Delete(ctx, projectionName, esdb.DeleteProjectionOptions{})
	if err != nil {
		return fmt.Errorf("刪除 projection 失敗: %w", err)
	}

	return nil
}

func isProjectionConflictError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "Conflict")
}
