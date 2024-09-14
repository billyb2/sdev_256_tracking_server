package main

import (
	"fmt"

	"github.com/billyb2/tracking_server/api"
	dblib "github.com/billyb2/tracking_server/db"
	_ "github.com/billyb2/tracking_server/docs"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title		Tracking Server API
//
// @BasePath	/api
func main() {
	db, err := dblib.NewDBConnection()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		dblib.WithGinContext(c, db)
	})

	v1 := r.Group("/api")
	v1.POST("/register", api.Register)
	v1.POST("/login", api.Login)
	v1.POST("/start_tracking", api.StartTrackingGroups)
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	fmt.Println("Starting server!")
	r.Run(":8080")
}
