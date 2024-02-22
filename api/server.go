package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/service"
	"github.com/techschool/simplebank/token"
	"github.com/techschool/simplebank/util"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
	srv        *http.Server
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	router := server.setupRouter(config, store, tokenMaker)
	server.router = router

	srv := &http.Server{
		Addr:    config.HTTPServerAddress,
		Handler: router,
	}
	server.srv = srv

	return server, nil
}

func (server *Server) setupRouter(config util.Config, store db.Store, tokenMaker token.Maker) *gin.Engine {
	router := gin.Default()

	userService := service.NewUserService(config, store, tokenMaker)
	userHandler := NewUserHandler(&userService)

	router.POST("/users", userHandler.createUser)
	router.POST("/users/login", userHandler.loginUser)
	router.POST("/tokens/renew_access", server.renewAccessToken)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccount)
	authRoutes.GET("/accounts", server.listAccounts)

	authRoutes.POST("/transfers", server.createTransfer)

	return router
}

// Start runs the HTTP server on a specific address.
//
//	func (server *Server) Start(address string) error {
//		return server.router.Run(address)
//	}
func (server *Server) Start() error {
	return server.srv.ListenAndServe()
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.srv.Shutdown(ctx)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
