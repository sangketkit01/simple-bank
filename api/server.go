package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/token"
	"github.com/sangketkit01/simple-bank/util"
)

// Server serves HTTP requests for our banking service.
type Server struct{
	config util.Config
	store db.Store
	tokenMaker token.Maker
	router *gin.Engine
}

// NewServer creates a new HTTP server and setup routing
func NewServer(config util.Config, store db.Store) (*Server, error){
	tokerMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil{
		return nil, fmt.Errorf("cannot create token maker: %w",err)
	}
	server := &Server{
		config: config,
		store: store,
		tokenMaker: tokerMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate) ; ok{
		v.RegisterValidation("currency",validCurrency)
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter(){
	router := gin.Default()
	router.POST("/users",server.createUser)
	router.POST("/users/login",server.loginUser)
	router.POST("/tokens/renew_access",server.renewAccessToken)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

	authRoutes.POST("/accounts",server.createAccount)
	authRoutes.GET("/accounts/:id",server.getAccount)
	authRoutes.GET("/accounts",server.listAccount)

	authRoutes.POST("/transfers",server.createTransfer)

	server.router = router
}

// Start tuns the HTTP server on a specific address
func (server *Server) Start(address string) error{
	return server.router.Run(address)
}

func errorResponse(err error) gin.H{
	return gin.H{"error":err.Error()}
}