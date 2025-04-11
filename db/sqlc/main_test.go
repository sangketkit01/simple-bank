package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

var testQueries *Queries
const(
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5433/simple_bank?sslmode=disable"
)

func TestMain(m *testing.M) {
	conn, err := sql.Open(dbDriver,dbSource)
	if err != nil{
		log.Fatalln("cannot connect to the database:",err)
	}

	testQueries = New(conn)

	os.Exit(m.Run())
}