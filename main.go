package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/sangketkit01/simple-bank/api"
	db "github.com/sangketkit01/simple-bank/db/sqlc"
	"github.com/sangketkit01/simple-bank/util"
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
	server, err := api.NewServer(config, store)
	if err != nil{
		log.Fatalln("cannot create server:",err)
	}

	err = server.Start(config.ServerAddress)
	if err != nil{
		log.Fatalln("cannot start server:",err)
	}

}