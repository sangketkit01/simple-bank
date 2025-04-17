package main

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/sangketkit01/simple-bank/api"
	apigrpc "github.com/sangketkit01/simple-bank/api_grpc"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	_ "github.com/sangketkit01/simple-bank/doc/statik"
	"github.com/sangketkit01/simple-bank/pb"
	"github.com/sangketkit01/simple-bank/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)


func main() {
	config, err := util.LoadConfig(".")
	if err != nil{
		log.Fatal().Msgf("cannot load config: %s",err)
	}

	if config.Environment == "development"{
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	conn, err := sql.Open(config.DBDriver,config.DBSource)
	if err != nil{
		log.Fatal().Msgf("cannot connect to database: %s",err)
	}

	// run db migration
	runDBMigration(config.MigrationUrl, config.DBSource)

	store := db.NewStore(conn)
	go runGatewayServer(config, store)
	runGrpcServer(config, store)

}

func runDBMigration(migrationURL string, dbSource string){
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil{
		log.Error().Msgf("cannot create new migrate instance: %s",err)
	}

	if err = migration.Up() ; err != nil && err != migrate.ErrNoChange{
		log.Error().Msgf("failed to run migrate up: %s",err)
	}

	log.Info().Msg("db migrated successfully")
}

func runGrpcServer(config util.Config, store db.Store){
	server, err := apigrpc.NewServer(config, store)
	if err != nil{
		log.Error().Msgf("cannot create server: %s",err)
	}

	grpcLogger := grpc.UnaryInterceptor(apigrpc.GrpcLogger)
	grpcServer := grpc.NewServer(grpcLogger)

	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp",config.GrpcServerAddress)
	if err != nil{
		log.Error().Msgf("cannot create listener: %s",err)
	}

	log.Info().Msgf("start gRPC server at %s",listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil{
		log.Error().Msgf("cannot start grpc server: %s", err)
	}

	log.Info().Msg("server started!")
}

func runGatewayServer(config util.Config, store db.Store){
	server, err := apigrpc.NewServer(config, store)
	if err != nil{
		log.Error().Msgf("cannot create server: %s",err)
	}
	
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})
	grpcMux := runtime.NewServeMux(jsonOption)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil{
		log.Error().Msgf("cannot register handler server: %s",err)
	}

	mux := http.NewServeMux()
	mux.Handle("/",grpcMux)

	statikFS, err := fs.New()
	if err != nil{
		log.Error().Msgf("cannot create statik fs: %s",err)
	}

	swaggerHandler := http.StripPrefix("/swagger/",http.FileServer(statikFS))
	mux.Handle("/swagger/",swaggerHandler)

	listener, err := net.Listen("tcp",config.HttpServerAddress)
	if err != nil{
		log.Error().Msgf("cannot create listener: %s",err)
	}

	log.Info().Msgf("start HTTP gateway server at %s",listener.Addr().String())
	handler := apigrpc.HttpLogger(mux)
	err = http.Serve(listener, handler)
	if err != nil{
		log.Error().Msgf("cannot start HTTP gateway server: %s", err)
	}

	log.Info().Msg("server started!")
}

func runGinServer(config util.Config, store db.Store ){
	server, err := api.NewServer(config, store)
	if err != nil{
		log.Error().Msgf("cannot create server: %s",err)
	}

	err = server.Start(config.HttpServerAddress)
	if err != nil{
		log.Error().Msgf("cannot start server: %s",err)
	}
}