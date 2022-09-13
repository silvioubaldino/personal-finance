package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenGORMConnection(source string) *gorm.DB {
	postgresConn := OpenPostgresConnection(source)
	fmt.Println("will open gorm conn")
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: postgresConn,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("could not create gorm connection: %s", err)
	}
	return gormDB
}
