package middleware

import (
	"context"
	"net/http"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/google/uuid"
)

func RequestIdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//從header內檢查是否有"request_id"
		requestId := r.Header.Get("request_id")
		if requestId == "" {
			requestId = uuid.New().String()
		}

		// 將資訊存儲到請求上下文
		ctx := context.WithValue(r.Context(), constants.RequestIDKey, requestId)

		// 使用更新後的上下文繼續請求
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
