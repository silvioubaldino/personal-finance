package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func OpenGORMConnection(source string) *gorm.DB {
	postgresConn := OpenPostgresConnection(source)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: postgresConn,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("could not create gorm connection: %s", err)
	}
	return gormDB
}
