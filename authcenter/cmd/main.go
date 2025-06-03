package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoyceAzure/lab/authcenter/internal/api"
	"github.com/RoyceAzure/lab/authcenter/internal/api/handler"
	"github.com/RoyceAzure/lab/authcenter/internal/api/router"
	"github.com/RoyceAzure/lab/authcenter/internal/appcontext"
	"github.com/RoyceAzure/lab/authcenter/internal/config"
)

// @title authcenter
// @version 1.0
// @description 通用認證中心
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        Authorization
// @description                 Description for Authorization header: Type "Bearer" followed by a space and the token. Example: "Bearer {token}"

func main() {
	// log.Printf("start scheduler")
	// zerolog.TimeFieldFormat = time.RFC3339

	app, err := appcontext.NewApplicationContext(config.GetConfig())
	if err != nil {
		log.Fatal(err)
		return
	}

	// 初始化 handler
	authHandler := handler.NewAuthHandler(app.AuthService, app.UserService)

	server := api.NewServer(authHandler)

	// 設置路由
	r := router.SetupRouter(server, app)

	// 設定服務器參數
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", app.Cf.ServerPort),
		Handler: r,
	}

	// 設置訊號監聽
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	shutDonwCompleted := make(chan struct{}, 1)
	// 監聽退出訊號
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		if err := app.Shutdown(shutdownCtx); err != nil {
			log.Printf("Application shutdown error: %v", err)
		}

		shutDonwCompleted <- struct{}{}
	}()

	// 啟動服務
	log.Printf("Server starting on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-shutDonwCompleted
	log.Printf("closed completed")
}
