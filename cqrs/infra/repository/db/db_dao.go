package db

import (
	"github.com/RoyceAzure/lab/cqrs/infra/repository/db/model"
	"gorm.io/gorm"
)

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
