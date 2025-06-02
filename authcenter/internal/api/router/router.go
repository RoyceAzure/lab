package router

import (
	"fmt"
	"net/http"

	_ "github.com/RoyceAzure/lab/authcenter/docs"
	"github.com/RoyceAzure/lab/authcenter/internal/api"
	m "github.com/RoyceAzure/lab/authcenter/internal/api/middleware"
	"github.com/RoyceAzure/rj/api/token"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRouter(server *api.Server, tokenMaker token.Maker[uuid.UUID], logger *zerolog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// c := cors.New(cors.Options{
	// 	AllowedOrigins:   []string{"http://localhost:3000"}, // 允許來自 localhost:3000 的請求
	// 	AllowCredentials: true,
	// 	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},           // 允許的 HTTP 方法
	// 	AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"}, // 允許的頭部
	// })

	// 全局中間件

	r.Use(m.RequestIdMiddleware)
	r.Use(m.AuthPayloadMiddleware(tokenMaker))
	r.Use(middleware.RealIP)
	r.Use(m.DeviceInfoMiddleware)
	r.Use(m.LoggerMiddleware(logger))
	// 配置 CORS

	// Swagger 文檔
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler())

	// API 路由
	r.Route("/api/v1", func(r chi.Router) {
		//Auth相關路由
		r.Group(func(r chi.Router) {
			r.Route("/auth", func(r chi.Router) {
				r.Post("/login/google", server.AuthHandler.GoogleLogin)
				r.Post("/login/account", server.AuthHandler.AccountAndPasswordLogin)
				r.Post("/refresh-token", server.AuthHandler.ReNewToken)
				r.Post("/logout", server.AuthHandler.LogOut)
				r.Post("/create-vertify-email", server.AuthHandler.CreateVertifyUserEmailLink)
				r.Get("/vertify-email", server.AuthHandler.VertifyUserEmailLink)
				r.Post("/linkedUser", server.AuthHandler.LinkedUserAccountAndPassword)
				r.With(m.AuthMiddleware).Get("/me", server.AuthHandler.Me)
				r.With(m.AuthMiddleware).Get("/permissions", server.AuthHandler.Permissions)
			})
		})
	})
	// 在設置完所有路由後打印路由樹
	fmt.Println(chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("%s %s\n", method, route)
		return nil
	}))
	return r
}
