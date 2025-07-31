package redis_decorator

import (
	"context"
	"time"

	"reflect"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/producer"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/db"
	"github.com/RoyceAzure/lab/cqrs/internal/infra/repository/redis_repo"
	"github.com/rs/zerolog/log"
)

/*
redis 專注商品庫存，所以只有跟商品庫存有關操作，才需要連動redis
*/
type CacheAsideProductRepo struct {
	db.IProductRepository
	redis redis_repo.IProductRedisRepository
}

func NewCacheAsideProductRepo(db db.IProductRepository, redis redis_repo.IProductRedisRepository) db.IProductRepository {
	return &CacheAsideProductRepo{IProductRepository: db, redis: redis}
}

func (p *CacheAsideProductRepo) CreateProduct(ctx context.Context, product *model.Product) error {
	err := p.IProductRepository.CreateProduct(ctx, product)
	if err != nil {
		return err
	}
	err = p.redis.CreateProductStock(ctx, product.ProductID, product.Stock)
	if err != nil {
		return err
	}
	return nil
}

func (p *CacheAsideProductRepo) UpdateProduct(ctx context.Context, product *model.Product) error {
	err := p.IProductRepository.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	err = p.redis.UpdateProductStock(ctx, product.ProductID, product.Stock)
	if err != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			p.redis.UpdateProductStock(ctx, product.ProductID, product.Stock)
		}()
		return err
	}
	return nil
}

func (p *CacheAsideProductRepo) UpdateStock(ctx context.Context, id string, stock uint) error {
	err := p.IProductRepository.UpdateStock(ctx, id, stock)
	if err != nil {
		return err
	}

	err = p.redis.UpdateProductStock(ctx, id, stock)
	if err != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			p.redis.UpdateProductStock(ctx, id, stock)
		}()
		return err
	}
	return nil
}

func (p *CacheAsideProductRepo) HardDeleteProduct(ctx context.Context, id string) error {
	err := p.IProductRepository.HardDeleteProduct(ctx, id)
	if err != nil {
		return err
	}

	err = p.redis.DeleteProductStock(context.Background(), id)
	if err != nil {
		go func() {
			time.Sleep(500 * time.Millisecond)
			p.redis.DeleteProductStock(context.Background(), id)
		}()
		return err
	}
	return nil
}

/*
優先修改redis cache 資料，並使用異步方式修改db資料

注意  db 跟 redis 儲存資料不同  這裡是特殊用法  必非傳統WiteBack模式
redis 只儲存庫存資訊 詳細資訊存在db
目前只需要增強 CreateProduct , AddProductStock 和 DeductProductStock 行為
*/
type WiteBackProductRepo struct {
	producer *producer.ProductProducer
	redis_repo.IProductRedisRepository
	db.IProductRepository
}

func NewWiteBackProductRepo(producer *producer.ProductProducer, redisRepo redis_repo.IProductRedisRepository, dbRepo db.IProductRepository) *WiteBackProductRepo {
	if producer == nil {
		panic("NewWiteBackProductRepo: producer cannot be nil")
	}

	v1 := reflect.ValueOf(redisRepo)
	if v1.Kind() == reflect.Ptr && v1.IsNil() {
		panic("NewWiteBackProductRepo: redis repository implementation cannot be nil")
	}

	v2 := reflect.ValueOf(dbRepo)
	if v2.Kind() == reflect.Ptr && v2.IsNil() {
		panic("NewWiteBackProductRepo: db repository implementation cannot be nil")
	}

	return &WiteBackProductRepo{
		producer:                producer,
		IProductRedisRepository: redisRepo,
		IProductRepository:      dbRepo,
	}
}

func (w *WiteBackProductRepo) CreateProduct(ctx context.Context, product *model.Product) error {
	err := w.IProductRedisRepository.CreateProductStock(ctx, product.ProductID, product.Stock)
	if err != nil {
		return err
	}
	go func() {
		if err := w.producer.AddNewProduct(ctx, product); err != nil {
			log.Error().Err(err).Msgf("kafka add new product produce failed, product_id with stock quantity: %s, %d", product.ProductID, product.Stock)
		}
	}()

	return nil
}

func (w *WiteBackProductRepo) GetProductStock(ctx context.Context, productID string) (int, error) {
	stock, err := w.IProductRedisRepository.GetProductStock(ctx, productID)
	if err != nil {
		return w.IProductRepository.GetProductStock(ctx, productID)
	}

	return stock, nil
}

func (w *WiteBackProductRepo) AddProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	stock, timestamp, err := w.IProductRedisRepository.AddProductStock(ctx, productID, quantity)
	if err != nil {
		return 0, err
	}

	go func() {
		if err := w.producer.UpdateProductReserved(ctx, &model.Product{ProductID: productID, Reserved: uint(stock)}, timestamp); err != nil {
			// 監控告警
			// metrics.IncrKafkaFailure()
			log.Error().Err(err).Msgf("kafka add product stock produce failed, product_id with stock quantity: %s, %d", productID, stock)
		}
	}()

	return stock, nil
}

func (w *WiteBackProductRepo) DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	stock, timestamp, err := w.IProductRedisRepository.DeductProductStock(ctx, productID, quantity)
	if err != nil {
		return 0, err
	}

	go func() {
		if err := w.producer.UpdateProductReserved(ctx, &model.Product{ProductID: productID, Reserved: uint(stock)}, timestamp); err != nil {
			// 監控告警
			// metrics.IncrKafkaFailure()
			log.Error().Err(err).Msgf("kafka deduct product stock produce failed, product_id with stock quantity: %s, %d", productID, stock)
		}
	}()
	return stock, nil
}

func (w *WiteBackProductRepo) HardDeleteProduct(ctx context.Context, productID string) error {
	_, _, err := w.IProductRedisRepository.DeductProductStock(ctx, productID, 0)
	if err != nil {
		return err
	}

	go func() {
		if err := w.producer.DeleteProduct(ctx, productID); err != nil {
			// 監控告警
			// metrics.IncrKafkaFailure()
			log.Error().Err(err).Msgf("kafka delete product produce failed, product_id: %s", productID)
		}
	}()
	return nil
}

var _ db.IProductRepository = &WiteBackProductRepo{}
