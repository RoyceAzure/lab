package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type IAuthorizer interface {
	AuthorizUser(ctx context.Context) (*token.Payload[uuid.UUID], string, error)
}

type Authorizer struct {
	tokenMaker token.Maker[uuid.UUID]
}

func NewAuthorizor(tokenMaker token.Maker[uuid.UUID]) IAuthorizer {
	return &Authorizer{
		tokenMaker: tokenMaker,
	}
}

/*
從ctx 取得metadata
從metadata取得authHeader => 檢查格式 => 檢查Bearer => 用tokenMaker 檢查token 內容
*/
func (auth *Authorizer) AuthorizUser(ctx context.Context) (*token.Payload[uuid.UUID], string, error) {
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
	payload, err := auth.tokenMaker.VertifyToken(accessToken)
	if err != nil {
		return nil, "", fmt.Errorf("invaliad access token : %s", err)
	}
	return payload, accessToken, nil
}
