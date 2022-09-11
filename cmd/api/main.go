package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/api"
	"personal-finance/internal/business/service/category"
	"personal-finance/internal/plataform/database"
	"personal-finance/internal/repositories/categories"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()

	dataSourceName := "postgres://admin:admin@localhost:5432/personal_finance?sslmode=disable"
	db := database.OpenGORMConnection(dataSourceName)
	repo := categories.PgRepository{Gorm: db}
	service := category.NewService(repo)
	api.AddHandlers(r, service)

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
