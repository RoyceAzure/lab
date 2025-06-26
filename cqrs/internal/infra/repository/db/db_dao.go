package db

import (
	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"gorm.io/gorm"
)

// 特定repo需要加上resdis操做
type DbDao struct {
	*gorm.DB
}

func NewDbDao(conn *gorm.DB) *DbDao {
	return &DbDao{
		DB: conn,
	}
}

// 初始化db schema
// 冪等性
func (d *DbDao) InitMigrate() error {
	return d.AutoMigrate(
		&model.User{},
		&model.Product{},
		&model.Order{},
		&model.OrderItem{},
	)
}
