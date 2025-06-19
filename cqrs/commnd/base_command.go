package command

type BaseCommand struct {
	CommandID string `json:"command_id"`
}
type CommandType string

const (
	OrderCreatedCommandName   CommandType = "OrderCreated"
	OrderConfirmedCommandName CommandType = "OrderConfirmed"
	OrderShippedCommandName   CommandType = "OrderShipped"
	OrderCancelledCommandName CommandType = "OrderCancelled"
	OrderRefundedCommandName  CommandType = "OrderRefunded"
)

type Command interface {
	Type() CommandType
}
