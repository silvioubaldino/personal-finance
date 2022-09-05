package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error running app: %v", err)
	}
}

func run() error {
	r := gin.Default()

	//db := make(map[string]model.Car)
	//repo := cars.NewMemoryRepository(db)
	//idGen := idgenerator.NewUUIDGenerator()
	//svc := car.NewService(repo, idGen)
	//api.AddHandlers(r, svc)

	if err := r.Run(); err != nil {
		return fmt.Errorf("error running web application: %v", err)
	}
	return nil
}
