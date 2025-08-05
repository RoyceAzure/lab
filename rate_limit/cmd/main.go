package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/RoyceAzure/lab/rate_limit/internal/alogrithm"
	"github.com/RoyceAzure/lab/rate_limit/limiter"
	"github.com/RoyceAzure/lab/rate_limit/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	r := chi.NewRouter()

	// 基本的 Chi 中間件
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// 本地限流配置
	//以下配置為10000RPS，每100ms補充15個token
	localConfig := &alogrithm.LimiterConfig{
		Capacity:    9000,
		RatePS:      9000,
		RatePPeriod: 90,
		RefillRate:  10 * time.Millisecond,
	}

	redisConfig := &limiter.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "password",
	}

	// API 路由分組
	r.Route("/api", func(r chi.Router) {
		// 本地限流的API組
		r.Group(func(r chi.Router) {
			// 使用固定窗口限流器
			r.Use(middleware.NewRateLimitMiddleware(middleware.FixedWindow, localConfig))
			r.Get("/fixed-window", handleRequest("Fixed Window API"))
		})

		r.Group(func(r chi.Router) {
			// 使用令牌桶限流器
			r.Use(middleware.NewRateLimitMiddleware(middleware.TokenBucket, localConfig))
			r.Get("/token-bucket", handleRequest("Token Bucket API"))
		})

		r.Group(func(r chi.Router) {
			// 使用滑動窗口限流器
			r.Use(middleware.NewRateLimitMiddleware(middleware.SlideWindow, localConfig))
			r.Get("/slide-window", handleRequest("Slide Window API"))
		})

		// Redis分散式限流的API組
		r.Group(func(r chi.Router) {
			// 使用Redis令牌桶限流器
			r.Use(middleware.NewRateLimitMiddleware(middleware.RedisBucket, localConfig, limiter.WithRedisConfig(redisConfig)))
			r.Get("/redis-bucket", handleRequest("Redis Bucket API"))
		})

		// 測試端點組
		r.Route("/test", func(r chi.Router) {
			// 快速請求測試
			r.Get("/burst", handleBurstTest())
			// 並發請求測試
			r.Get("/concurrent", handleConcurrentTest())
		})
	})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func handleRequest(apiName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Message:   apiName + " response",
			Timestamp: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleBurstTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		responses := make([]Response, 0)

		// 快速發送10個請求
		for i := 0; i < 10; i++ {
			responses = append(responses, Response{
				Message:   "Burst test response " + string(rune(i+'0')),
				Timestamp: time.Now(),
			})
			time.Sleep(100 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}
}

func handleConcurrentTest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		responses := make([]Response, 0)
		done := make(chan Response, 10)

		// 並發發送10個請求
		for i := 0; i < 10; i++ {
			go func(i int) {
				done <- Response{
					Message:   "Concurrent test response " + string(rune(i+'0')),
					Timestamp: time.Now(),
				}
			}(i)
		}

		// 收集所有響應
		for i := 0; i < 10; i++ {
			responses = append(responses, <-done)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}
}
