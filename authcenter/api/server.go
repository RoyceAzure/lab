package api

import "github.com/RoyceAzure/lab/authcenter/api/handler"

type Server struct {
	AuthHandler *handler.AuthHandler
}

func NewServer(
	authHandler *handler.AuthHandler,
) *Server {
	return &Server{
		AuthHandler: authHandler,
	}
}
