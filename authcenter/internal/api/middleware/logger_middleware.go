package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/RoyceAzure/lab/authcenter/internal/constants"
	"github.com/RoyceAzure/lab/authcenter/internal/util"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type StatusRecoder struct {
	http.ResponseWriter
	status int
}

func (w *StatusRecoder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *StatusRecoder) Status() int {
	return w.status
}

func getRequestID(r *http.Request) string {
	requestId := "unknown"
	if v := r.Context().Value(constants.RequestIDKey); v != nil {
		requestId = v.(string)
	}
	return requestId
}

func getPayload(r *http.Request) *token.Payload[uuid.UUID] {
	ctx := r.Context()
	payload := util.GetTokenPayloadFromContext[uuid.UUID](ctx)
	if payload == nil {
		return &token.Payload[uuid.UUID]{
			UPN:    "unknown",
			UserId: uuid.Nil,
		}
	}
	return payload
}

// 記錄request 請求
// 有一起處理recover
func LoggerMiddleware(logger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recoder := &StatusRecoder{
				ResponseWriter: w,
			}
			next.ServeHTTP(recoder, r)

			requestId := getRequestID(r)
			payload := getPayload(r)

			if logger == nil {
				temp := zerolog.New(os.Stdout).With().Timestamp().Logger()
				logger = &temp
			}

			logger.Info().
				Str("request_id", requestId).
				Str("upn", payload.UPN).
				Str("user_id", payload.UserId.String()).
				Str("method", r.Method).
				Str("url", r.URL.String()).
				Int("status", recoder.Status()).
				Msg("request completed")

			defer func() {
				if err := recover(); err != nil {
					requestId := getRequestID(r)
					payload := getPayload(r)

					var errMsg string
					if e, ok := err.(error); ok {
						errMsg = e.Error()
					} else {
						errMsg = fmt.Sprintf("%v", err)
					}
					// status 建議用自訂 ResponseWriter 取得
					logger.Error().
						Str("request_id", requestId).
						Str("upn", payload.UPN).
						Str("user_id", payload.UserId.String()).
						Str("method", r.Method).
						Str("url", r.URL.String()).
						Int("status", recoder.Status()).
						Str("error", errMsg).
						Msg("request completed")

					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Internal Server Error",
					})
				}
			}()
		})
	}
}
