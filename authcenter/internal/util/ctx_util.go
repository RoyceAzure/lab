package util

import (
	"context"
	"net"
	"net/netip"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/rj/api/token"
)

type DeviceInfo struct {
	UserAgent  string     // 使用者代理字符串
	IPAddress  netip.Addr // IP地址
	Region     string     // 地區資訊
	DeviceType string     // 設備類型資訊
}

// GetDeviceInfoFromContext 從請求上下文中獲取設備相關資訊
//
// 參數:
//   - ctx: 包含設備資訊的上下文
//
// 返回值:
//   - DeviceInfo: 包含了所有設備相關資訊的結構體
//
// 錯誤:
//   - 該函數不返回錯誤，若資訊不存在或IP地址解析失敗則相應字段為空值
//
// 範例:
//
//	deviceInfo := GetDeviceInfoFromContext(r.Context())
func GetDeviceInfoFromContext(ctx context.Context) DeviceInfo {
	var info DeviceInfo

	if ua := ctx.Value(constants.AuthorizationUserAgentKey); ua != nil {
		info.UserAgent = ua.(string)
	}

	if ip := ctx.Value(constants.AuthorizationIPKey); ip != nil {
		ipStr := ip.(string)
		// 移除端口部分（如果有）
		if host, _, err := net.SplitHostPort(ipStr); err == nil {
			ipStr = host
		}
		// 嘗試解析IP地址
		if addr, err := netip.ParseAddr(ipStr); err == nil {
			info.IPAddress = addr
		}
	}

	if r := ctx.Value(constants.AuthorizationRegionKey); r != nil {
		info.Region = r.(string)
	}

	if di := ctx.Value(constants.AuthorizationDeviceInfoKey); di != nil {
		info.DeviceType = di.(string)
	}

	return info
}

func GetTokenPayloadFromContext[T token.UserIDConstraint](ctx context.Context) *token.Payload[T] {
	var tokenPayload *token.Payload[T]

	if v := ctx.Value(constants.AuthorizationPayloadKey); v != nil {
		tokenPayload = v.(*token.Payload[T])
	}

	return tokenPayload
}
