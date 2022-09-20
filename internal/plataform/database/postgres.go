package database

import (
	"database/sql"
	"log"

	// import db driver.
	_ "github.com/lib/pq"
)

func OpenPostgresConnection(source string) *sql.DB {
	db, err := sql.Open("postgres", source)
	if err != nil {
		log.Fatalf("error to connect postgres: %s", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("could not ping the database: %s", err)
	}
	return db
}
