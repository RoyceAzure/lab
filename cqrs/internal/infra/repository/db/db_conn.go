package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetDbConn(dbname, host, port, user, pas string) (*gorm.DB, error) {
	// 資料來源名稱 (DSN)
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable", user, pas, host, port, dbname)

	// 連線到資料庫
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
