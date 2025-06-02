package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
)

func DeviceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 獲取 User-Agent
		userAgent := r.Header.Get("User-Agent")

		ipAddress := r.RemoteAddr

		region := "未知" // 預設值，實際使用時應替換

		deviceInfo := "未知"
		if strings.Contains(strings.ToLower(userAgent), "mobile") {
			deviceInfo = "移動裝置"
		} else if strings.Contains(strings.ToLower(userAgent), "tablet") {
			deviceInfo = "平板裝置"
		} else {
			deviceInfo = "桌面裝置"
		}

		// 將資訊存儲到請求上下文
		ctx := context.WithValue(r.Context(), constants.AuthorizationUserAgentKey, userAgent)
		ctx = context.WithValue(ctx, constants.AuthorizationIPKey, ipAddress)
		ctx = context.WithValue(ctx, constants.AuthorizationRegionKey, region)
		ctx = context.WithValue(ctx, constants.AuthorizationDeviceInfoKey, deviceInfo)

		// 使用更新後的上下文繼續請求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
