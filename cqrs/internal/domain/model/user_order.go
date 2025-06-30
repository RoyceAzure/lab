package model

type UserOrder struct {
	ID      int    `gorm:"primaryKey;type:int" json:"id"`
	UserID  int    `gorm:"not null;type:int" json:"user_id"`
	OrderID string `gorm:"not null;type:varchar(255)" json:"order_id"`
	BaseModel
}
