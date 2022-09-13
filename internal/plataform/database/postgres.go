package database

import (
	"database/sql"
	"fmt"
	"log"

	// import db driver.
	_ "github.com/lib/pq"
)

func OpenPostgresConnection(source string) *sql.DB {
	fmt.Println("will open pg conn")
	db, err := sql.Open("postgres", source)
	if err != nil {
		log.Fatalf("error to connect postgres: %s", err)
	}
	fmt.Println("will ping pg conn")
	if err := db.Ping(); err != nil {
		log.Fatalf("could not ping the database: %s", err)
	}
	fmt.Println("success ping pg conn")
	return db
}
