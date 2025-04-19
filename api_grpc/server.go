package apigrpc

import (
	"fmt"

	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/token"
	"github.com/sangketkit01/simple-bank/util"
	"github.com/sangketkit01/simple-bank/worker"
)

// Server serves HTTP requests for our banking service.
type Server struct{
	pb.UnimplementedSimpleBankServer
	config util.Config
	store db.Store
	tokenMaker token.Maker
	taskDistributor worker.TaskDistributor
}

// NewServer creates a new gRPC server and setup routing
func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error){
	tokerMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil{
		return nil, fmt.Errorf("cannot create token maker: %w",err)
	}
	server := &Server{
		config: config,
		store: store,
		tokenMaker: tokerMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}




