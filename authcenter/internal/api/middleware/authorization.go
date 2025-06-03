package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/lab/rj_redis/pkg/cache"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type IAuthorizer interface {
	AuthorizUser(ctx context.Context) (*token.Payload[uuid.UUID], string, error)
	AuthPayloadMiddlewareFunc() func(http.Handler) http.Handler
}

type Authorizer struct {
	tokenMaker token.Maker[uuid.UUID]
	cache      cache.Cache
}

func NewAuthorizor(tokenMaker token.Maker[uuid.UUID], cache cache.Cache) IAuthorizer {
	return &Authorizer{
		tokenMaker: tokenMaker,
		cache:      cache,
	}
}

/*
從ctx 取得metadata
從metadata取得authHeader => 檢查格式 => 檢查Bearer => 用tokenMaker 檢查token 內容
*/
func (a *Authorizer) AuthorizUser(ctx context.Context) (*token.Payload[uuid.UUID], string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, "", fmt.Errorf("misssing metadata")
	}

	values := md.Get(string(constants.AuthorizationHeaderKey))
	if len(values) == 0 {
		return nil, "", fmt.Errorf("misssing authorization header")
	}
	authHeader := strings.Fields(values[0])
	if len(authHeader) != 2 {
		return nil, "", fmt.Errorf("invalid auth format")
	}
	authType := strings.ToLower(authHeader[0])
	if authType != string(constants.AuthorizationTypeBearer) {
		return nil, "", fmt.Errorf("unsportted authorization type : %s", authType)
	}

	accessToken := authHeader[1]
	payload, err := a.tokenMaker.VertifyToken(accessToken)
	if err != nil {
		return nil, "", fmt.Errorf("invaliad access token : %s", err)
	}
	return payload, accessToken, nil
}

// 驗證token 但若token以任何錯誤 都不會中斷，這裡僅做解析token payload, 若payload有錯誤，則不會設置context
func (a *Authorizer) AuthPayloadMiddlewareFunc() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, ok := a.checkTokenAndPayload(r)
			if ok {
				ctx := context.WithValue(r.Context(), constants.AuthorizationPayloadKey, payload)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func (a *Authorizer) checkTokenAndPayload(r *http.Request) (*token.Payload[uuid.UUID], bool) {
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

	banned, err := a.cache.Exists(r.Context(), accessToken)
	if err != nil {
		return nil, false
	}
	if banned {
		return nil, false
	}

	payload, err := a.tokenMaker.VertifyToken(accessToken)
	if err != nil {
		return nil, false
	}

	return payload, true
}
