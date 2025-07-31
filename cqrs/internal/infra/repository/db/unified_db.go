package db

import (
	"context"

	"github.com/RoyceAzure/lab/cqrs/internal/domain/model"
	"gorm.io/gorm"
)

// UnifiedDB 統一的資料庫介面
type UnifiedDB interface {
	// 基礎操作
	GetDB() *gorm.DB
	Begin(ctx context.Context) *gorm.DB
	InitMigrate() error

	// Product 相關操作
	IProductRepository

	// Order 相關操作
	IOrderRepository

	// User 相關操作
	IUserRepository

	// UserOrder 相關操作
	IUserOrderRepository
}

// IProductRepository Product 相關操作介面
type IProductRepository interface {
	CreateProduct(ctx context.Context, product *model.Product) error
	GetProductByID(ctx context.Context, productID string) (*model.Product, error)
	GetProductByCode(ctx context.Context, code string) (*model.Product, error)
	GetAllProducts(ctx context.Context) ([]model.Product, error)
	UpdateProduct(ctx context.Context, product *model.Product) error
	UpdateStock(ctx context.Context, id string, stock uint) error
	UpdateReserved(ctx context.Context, id string, reserved uint) error
	HardDeleteProduct(ctx context.Context, id string) error
	GetProductsInStock(ctx context.Context) ([]model.Product, error)
	GetProductStock(ctx context.Context, productID string) (int, error)
	GetProductReserved(ctx context.Context, productID string) (int, error)
	AddProductStock(ctx context.Context, productID string, quantity uint) (int, error)
	DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error)
}

// IOrderRepository Order 相關操作介面
type IOrderRepository interface {
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrderByID(ctx context.Context, id string) (*model.Order, error)
	GetOrdersByUserID(ctx context.Context, userID int) ([]model.Order, error)
	GetAllOrders(ctx context.Context) ([]model.Order, error)
	UpdateOrder(ctx context.Context, order *model.Order) error
	UpdateOrderState(ctx context.Context, id string, state uint) error
	UpdateOrderAmount(ctx context.Context, id string, amount float64) error
	HardDeleteOrder(ctx context.Context, id string) error
}

// IUserRepository User 相關操作介面
type IUserRepository interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	GetUserByID(ctx context.Context, id int) (*model.User, error)
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id int) error
}

// IUserOrderRepository UserOrder 相關操作介面
type IUserOrderRepository interface {
	GetUserOrderByID(ctx context.Context, id int) (*model.UserOrder, error)
	ListUserOrders(ctx context.Context) ([]model.UserOrder, error)
	ListUserOrdersByUserID(ctx context.Context, userID int) ([]model.UserOrder, error)
	CreateUserOrder(ctx context.Context, userOrder *model.UserOrder) (*model.UserOrder, error)
	DeleteUserOrder(ctx context.Context, id int) error
}

// UnifiedDBImpl 統一資料庫實現
type UnifiedDBImpl struct {
	db    *gorm.DB
	dbDao *DbDao
	*ProductDBRepo
	*OrderRepo
	*UserRepo
	*UserOrderRepo
}

// NewUnifiedDB 創建新的統一資料庫實例
func NewUnifiedDB(db *gorm.DB) *UnifiedDBImpl {
	dbDao := NewDbDao(db)
	return &UnifiedDBImpl{
		db:            db,
		dbDao:         dbDao,
		ProductDBRepo: NewProductDBRepo(dbDao),
		OrderRepo:     NewOrderRepo(dbDao),
		UserRepo:      NewUserRepo(dbDao),
		UserOrderRepo: NewUserOrderRepo(dbDao),
	}
}

func (u *UnifiedDBImpl) InitMigrate() error {
	return u.dbDao.InitMigrate()
}

// GetDB 獲取資料庫連接
func (u *UnifiedDBImpl) GetDB() *gorm.DB {
	return u.db
}

// Begin 開始事務
func (u *UnifiedDBImpl) Begin(ctx context.Context) *gorm.DB {
	return u.db.WithContext(ctx).Begin()
}

// Product Repository 實現
func (u *UnifiedDBImpl) CreateProduct(ctx context.Context, product *model.Product) error {
	return u.ProductDBRepo.CreateProduct(ctx, product)
}

func (u *UnifiedDBImpl) GetProductByID(ctx context.Context, productID string) (*model.Product, error) {
	return u.ProductDBRepo.GetProductByID(ctx, productID)
}

func (u *UnifiedDBImpl) GetProductByCode(ctx context.Context, code string) (*model.Product, error) {
	return u.ProductDBRepo.GetProductByCode(ctx, code)
}

func (u *UnifiedDBImpl) AddProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	return u.ProductDBRepo.AddProductStock(ctx, productID, quantity)
}

func (u *UnifiedDBImpl) GetProductReserved(ctx context.Context, productID string) (int, error) {
	return u.ProductDBRepo.GetProductReserved(ctx, productID)
}

func (u *UnifiedDBImpl) DeductProductStock(ctx context.Context, productID string, quantity uint) (int, error) {
	return u.ProductDBRepo.DeductProductStock(ctx, productID, quantity)
}

func (u *UnifiedDBImpl) GetAllProducts(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	err := u.db.WithContext(ctx).Find(&products).Error
	return products, err
}

func (u *UnifiedDBImpl) UpdateReserved(ctx context.Context, id string, reserved uint) error {
	return u.ProductDBRepo.UpdateReserved(ctx, id, reserved)
}

func (u *UnifiedDBImpl) UpdateProduct(ctx context.Context, product *model.Product) error {
	return u.db.WithContext(ctx).Save(product).Error
}

func (u *UnifiedDBImpl) UpdateStock(ctx context.Context, id string, stock uint) error {
	return u.ProductDBRepo.UpdateStock(ctx, id, stock)
}

func (u *UnifiedDBImpl) HardDeleteProduct(ctx context.Context, id string) error {
	return u.db.WithContext(ctx).Unscoped().Delete(&model.Product{}, id).Error
}

func (u *UnifiedDBImpl) GetProductsInStock(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	err := u.db.WithContext(ctx).Where("stock > 0").Find(&products).Error
	return products, err
}

// Order Repository 實現
func (u *UnifiedDBImpl) CreateOrder(ctx context.Context, order *model.Order) error {
	return u.OrderRepo.CreateOrder(ctx, order)
}

func (u *UnifiedDBImpl) GetOrderByID(ctx context.Context, id string) (*model.Order, error) {
	return u.OrderRepo.GetOrderByID(ctx, id)
}

func (u *UnifiedDBImpl) GetOrdersByUserID(ctx context.Context, userID int) ([]model.Order, error) {
	return u.OrderRepo.GetOrdersByUserID(ctx, userID)
}

func (u *UnifiedDBImpl) GetAllOrders(ctx context.Context) ([]model.Order, error) {
	return u.OrderRepo.GetAllOrders(ctx)
}

func (u *UnifiedDBImpl) UpdateOrder(ctx context.Context, order *model.Order) error {
	return u.OrderRepo.UpdateOrder(ctx, order)
}

func (u *UnifiedDBImpl) UpdateOrderState(ctx context.Context, id string, state uint) error {
	return u.OrderRepo.UpdateOrderState(ctx, id, state)
}

func (u *UnifiedDBImpl) UpdateOrderAmount(ctx context.Context, id string, amount float64) error {
	return u.OrderRepo.UpdateOrderAmount(ctx, id, amount)
}

func (u *UnifiedDBImpl) HardDeleteOrder(ctx context.Context, id string) error {
	return u.OrderRepo.HardDeleteOrder(ctx, id)
}

// User Repository 實現
func (u *UnifiedDBImpl) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	return u.UserRepo.CreateUser(ctx, user)
}

func (u *UnifiedDBImpl) GetUserByID(ctx context.Context, id int) (*model.User, error) {
	return u.UserRepo.GetUserByID(ctx, id)
}

func (u *UnifiedDBImpl) GetAllUsers(ctx context.Context) ([]model.User, error) {
	return u.UserRepo.GetAllUsers(ctx)
}

func (u *UnifiedDBImpl) UpdateUser(ctx context.Context, user *model.User) error {
	return u.UserRepo.UpdateUser(ctx, user)
}

func (u *UnifiedDBImpl) DeleteUser(ctx context.Context, id int) error {
	return u.UserRepo.DeleteUser(ctx, id)
}

// UserOrder Repository 實現
func (u *UnifiedDBImpl) GetUserOrderByID(ctx context.Context, id int) (*model.UserOrder, error) {
	return u.UserOrderRepo.GetUserOrderByID(id)
}

func (u *UnifiedDBImpl) ListUserOrders(ctx context.Context) ([]model.UserOrder, error) {
	return u.UserOrderRepo.ListUserOrders()
}

func (u *UnifiedDBImpl) ListUserOrdersByUserID(ctx context.Context, userID int) ([]model.UserOrder, error) {
	return u.UserOrderRepo.ListUserOrdersByUserID(userID)
}

func (u *UnifiedDBImpl) CreateUserOrder(ctx context.Context, userOrder *model.UserOrder) (*model.UserOrder, error) {
	return u.UserOrderRepo.CreateUserOrder(userOrder)
}

func (u *UnifiedDBImpl) DeleteUserOrder(ctx context.Context, id int) error {
	return u.UserOrderRepo.DeleteUserOrder(id)
}

var (
	_ UnifiedDB            = (*UnifiedDBImpl)(nil)
	_ IProductRepository   = (*UnifiedDBImpl)(nil)
	_ IOrderRepository     = (*UnifiedDBImpl)(nil)
	_ IUserRepository      = (*UnifiedDBImpl)(nil)
	_ IUserOrderRepository = (*UnifiedDBImpl)(nil)
)
