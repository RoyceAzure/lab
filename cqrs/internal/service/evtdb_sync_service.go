package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	evt_model "github.com/RoyceAzure/lab/cqrs/internal/domain/model/event"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/eventdb"
	"github.com/RoyceAzure/lab/cqrs/internal/pkg/util"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type BackGroundService interface {
	Start() error
	Stop(timeout time.Duration) error
}

// sync order from evt db projection to order db table
type OrderSyncService struct {
	orderEventDBQueryDao *eventdb.EventDBQueryDao
	orderRepo            *db.OrderRepo
	orderList            []evt_model.OrderAggregate
	isOrderThreadRunning atomic.Bool
	stopCtxCancel        context.CancelFunc
}

func NewEventDBSyncService(orderEventDBQueryDao *eventdb.EventDBQueryDao, orderRepo *db.OrderRepo) *OrderSyncService {
	if orderEventDBQueryDao == nil {
		panic("evt sync service dependency orderEventDBQueryDao is nil")
	}
	if orderRepo == nil {
		panic("evt sync service dependency orderRepo is nil")
	}
	return &OrderSyncService{
		orderEventDBQueryDao: orderEventDBQueryDao,
		orderRepo:            orderRepo,
		orderList:            make([]evt_model.OrderAggregate, 0, 100),
		isOrderThreadRunning: atomic.Bool{},
	}
}

func (e *OrderSyncService) Start() error {
	if !e.isOrderThreadRunning.CompareAndSwap(false, true) {
		return fmt.Errorf("order sync thread is already running")
	}

	go func() {
		defer e.isOrderThreadRunning.CompareAndSwap(true, false)
		ctx, cancel := context.WithCancel(context.Background())
		e.stopCtxCancel = cancel
		defer e.stopCtxCancel()
		e.orderEventDBQueryDao.SubscribeOrderStream(ctx, e.batchUpdateOrderHandler)
	}()
	return nil
}

func (e *OrderSyncService) Stop(timeout time.Duration) error {
	if e.stopCtxCancel == nil {
		return nil
	}
	e.stopCtxCancel()

	time.AfterFunc(timeout, func() {
		if e.isOrderThreadRunning.Load() {
			log.Error().Msg("time out for stop order sync service")
		}
		e.isOrderThreadRunning.Store(false)
	})

	return nil
}

// 批次更新到order狀態
// 只有遇到gorm的事務錯誤，才會中斷流程
// TODO: 若遇到其他錯誤，需要紀錄到log，做後續補償
func (e *OrderSyncService) batchUpdateOrderHandler(event evt_model.OrderAggregate) error {
	e.orderList = append(e.orderList, event)
	result, err := e.flushOrder(1)
	if err != nil {
		return err
	}

	if result.FailCount > 0 {
		for key, failedOrder := range result.FailedOrders {
			fmt.Printf("failed to update order with id: %v, error: %v\n", key, failedOrder.Error())
		}
	}

	return nil
}

// BatchUpdateOrder
// 錯誤:
//
//	只有當錯誤是gorm的事務錯誤才回傳
//	其他錯誤不會中斷同步流程
func (e *OrderSyncService) flushOrder(flushCount int) (*db.BatchUpdateResult, error) {
	orderList := make([]*model.Order, flushCount)
	if len(e.orderList) >= flushCount {
		for i, orderAggreate := range e.orderList {
			orderList[i] = util.OrderAggregateToOrder(&orderAggreate)
		}
	}

	result, err := e.orderRepo.UpdateOrdersBatch(orderList)
	if err != nil {
		// 只有事務錯誤才回傳
		if checkGormTransactionError(err) {
			return nil, err
		}
	}

	e.orderList = e.orderList[flushCount:]

	return result, nil
}

func checkGormTransactionError(err error) bool {
	if errors.Is(err, gorm.ErrInvalidTransaction) || errors.Is(err, gorm.ErrNotImplemented) || errors.Is(err, gorm.ErrMissingWhereClause) {
		return true
	}
	return false
}
