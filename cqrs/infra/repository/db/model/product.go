package model

type Product struct {
	BaseModel               // 包含了 ID, CreatedAt, UpdatedAt, DeletedAt
	ProductID   uint        `gorm:"primaryKey"`
	Code        string      `gorm:"not null;type:varchar(100);unique"`
	Name        string      `gorm:"not null;type:varchar(100)"`
	Price       uint        `gorm:"not null;type:decimal(10,2)"`
	Stock       uint        `gorm:"not null;type:int"`
	Category    string      `gorm:"not null;type:varchar(50)"`
	Description string      `gorm:"not null;type:text"`
	OrderItems  []OrderItem `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"` // 一對多，級聯刪除
}
