package command

type CreateOrderCommand struct {
	UserID uint
	Items  []OrderItemCommand
}

type OrderItemCommand struct {
	ProductID uint
	Quantity  int
}

type AddProductCommand struct {
	OrderID   uint
	ProductID uint
	Quantity  int
}

type UpdateProductCommand struct {
	OrderID   uint
	ProductID uint
	Quantity  int
}

type DeleteProductCommand struct {
	OrderID   uint
	ProductID uint
}
