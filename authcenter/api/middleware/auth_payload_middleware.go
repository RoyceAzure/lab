package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/google/uuid"
)

// 驗證token 但若token以任何錯誤 都不會中斷，這裡僅做解析token payload, 若payload有錯誤，則不會設置context
func AuthPayloadMiddleware(tokenMaker token.Maker[uuid.UUID]) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := checkAuthPayload(tokenMaker, r)
			if ok {
				ctx := context.WithValue(r.Context(), constants.AuthorizationPayloadKey, payload)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func checkAuthPayload(tokenMaker token.Maker[uuid.UUID], r *http.Request) (*token.Payload[uuid.UUID], bool) {
	authorizationHeader := r.Header.Get(string(constants.AuthorizationHeaderKey))
	if len(authorizationHeader) == 0 {
		return nil, false
	}

	fields := strings.Fields(authorizationHeader)
	if len(fields) < 2 {
		return nil, false
	}

	authorizationType := strings.ToLower(fields[0])
	if authorizationType != string(constants.AuthorizationTypeBearer) {
		return nil, false
	}

	accessToken := fields[1]
	payload, err := tokenMaker.VertifyToken(accessToken)
	if err != nil {
		return nil, false
	}

	return payload, true
}
