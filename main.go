package main

import (
	"database/sql"
	"log"
	"net"

	_ "github.com/lib/pq"
	"github.com/sangketkit01/simple-bank/api"
	apigrpc "github.com/sangketkit01/simple-bank/api_grpc"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)


func main() {
	config, err := util.LoadConfig(".")
	if err != nil{
		log.Fatalln("cannot load config:",err)
	}

	conn, err := sql.Open(config.DBDriver,config.DBSource)
	if err != nil{
		log.Fatalln("cannot connect to database:",err)
	}

	store := db.NewStore(conn)
	runGrpcServer(config, store)

}

func runGrpcServer(config util.Config, store db.Store){
	server, err := apigrpc.NewServer(config, store)
	if err != nil{
		log.Fatalln("cannot create server:",err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp",config.GrpcServerAddress)
	if err != nil{
		log.Fatalln("cannot create listener:",err)
	}

	log.Printf("start gRPC server at %s",listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil{
		log.Fatalln("cannot start grpc server:", err)
	}
}

func runGinServer(config util.Config, store db.Store ){
	server, err := api.NewServer(config, store)
	if err != nil{
		log.Fatalln("cannot create server:",err)
	}

	err = server.Start(config.HttpServerAddress)
	if err != nil{
		log.Fatalln("cannot start server:",err)
	}
}