package middleware

import (
	"net/http"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/rj/api"
	"github.com/RoyceAzure/rj/api/token"
	er "github.com/RoyceAzure/rj/util/rj_error"
	"github.com/google/uuid"
)

//for gin
// func authMiddleware(tokenMaker token.Maker) http.Handler {
// 	return func(ctx *gin.Context) {
// 		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
// 		if len(authorizationHeader) < 0 {
// 			err := errors.New("authorization header is not provied")
// 			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
// 			return
// 		}

// 		fields := strings.Fields(authorizationHeader)
// 		if len(fields) < 2 {
// 			err := errors.New("invalid authorization header format")
// 			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
// 			return
// 		}

// 		authorizationType := strings.ToLower(fields[0])
// 		if authorizationType != authorizationTypeBearer {
// 			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
// 			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
// 			return
// 		}
// 		accessToken := fields[1]
// 		payload, err := tokenMaker.VertifyToken(accessToken)
// 		if err != nil {
// 			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
// 			return
// 		}

// 		ctx.Set(authorizationPayloadKey, payload)
// 		ctx.Next()
// 	}
// }

// 驗證是ctx是否有token payload
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value(constants.AuthorizationPayloadKey).(*token.Payload[uuid.UUID])
		if !ok {
			api.ErrorJSON(w, int(er.UnauthenticatedCode), er.New(er.UnauthenticatedCode, "unauthenticated"), er.ErrStrMap[er.UnauthenticatedCode])
			return
		}
		next.ServeHTTP(w, r)
	})
}
