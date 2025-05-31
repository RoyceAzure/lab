package constants

const (
	//分頁
	DefaultPagingSize int = 10
	DefaultPaging     int = 1
	TWSEHOSTURL           = "https://www.twse.com.tw/rwd/zh/afterTrading/STOCK_DAY"
)

type SortOrderEnum string

const (
	DefaultSortOrder SortOrderEnum = "asc"
	SortOrderAsc     SortOrderEnum = "asc"
	SortOrderDesc    SortOrderEnum = "desc"
)

func IsValidSortOrderEnum(order string) bool {
	switch SortOrderEnum(order) {
	case SortOrderAsc, SortOrderDesc:
		return true
	default:
		return false
	}
}

// for api auth
type ContextKey string

const (
	AuthorizationHeaderKey     ContextKey = "authorization"
	AuthorizationTypeBearer    ContextKey = "bearer"
	AuthorizationPayloadKey    ContextKey = "authorization_payload"
	AuthorizationUserAgentKey  ContextKey = "user_agent"
	AuthorizationIPKey         ContextKey = "ip_address"
	AuthorizationRegionKey     ContextKey = "region"
	AuthorizationDeviceInfoKey ContextKey = "device_info"
)

type TokenDurationHour int

const (
	AccessTokenDuration  TokenDurationHour = 24
	RefreshTokenDuration TokenDurationHour = 72
)

type ENV string

const (
	Debug ENV = "debug"
	Dev   ENV = "development"
	Stag  ENV = "staging"
	Prod  ENV = "production"
)

type RequestID string

const (
	RequestIDKey RequestID = "request_id"
)
