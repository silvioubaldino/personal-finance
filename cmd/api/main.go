package main

import (
	"fmt"
	"personal-finance/internal/domain/category/api"
	"personal-finance/internal/domain/category/repository"
	"personal-finance/internal/domain/category/service"

	"github.com/gin-gonic/gin"

	"personal-finance/internal/plataform/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()
	fmt.Println("starting")
	dataSourceName := "postgresql://admin:admin@pg-personal-finance:5432/personal_finance?sslmode=disable"
	db := database.OpenGORMConnection(dataSourceName)

	CategoryRepo := repository.NewPgRepository(db)
	CategoryService := service.NewCategoryService(CategoryRepo)

	api.NewCategoryHandlers(r, CategoryService)
	fmt.Println("connected")

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %w", err)
	}
	return nil
}
