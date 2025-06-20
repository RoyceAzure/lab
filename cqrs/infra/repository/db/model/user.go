package model

type User struct {
	UserID      int     `gorm:"primaryKey"`
	UserName    string  `gorm:"not null;type:varchar(50)"`
	UserEmail   string  `gorm:"unique;not null;type:varchar(50)"`
	UserPhone   string  `gorm:"unique;not null;type:varchar(50)"`
	UserAddress string  `gorm:"not null;type:varchar(255)"`
	Orders      []Order `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"` // 一對多，級聯刪除
	BaseModel
}
