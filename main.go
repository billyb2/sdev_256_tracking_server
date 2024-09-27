package main

import (
	"fmt"
	"os"
	"time"

	"github.com/billyb2/tracking_server/api"
	dblib "github.com/billyb2/tracking_server/db"
	_ "github.com/billyb2/tracking_server/docs"
	fedex "github.com/billyb2/tracking_server/tracking"
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

	go func() {
		for {
			rows, err := db.Query("select tracking_number from tracking where unixepoch('now', 'auto') - unixepoch(status_last_updated, 'auto') > 1800")
			if err != nil {
				fmt.Fprintln(os.Stderr, "error querying tracking numbers", err)
				continue
			}
			defer rows.Close()

			trackingNumbers := []string{}
			for rows.Next() {
				var trackingNumber string
				if err := rows.Scan(&trackingNumber); err != nil {
					fmt.Fprintln(os.Stderr, "error scanning fedex api", err)
					continue
				}
				trackingNumbers = append(trackingNumbers, trackingNumber)
			}

			trackingStatuses, err := fedex.TrackByTrackingNumber(trackingNumbers)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error calling fedex api", err)
				continue
			}
			tx, err := db.Begin()
			if err != nil {
				fmt.Fprintln(os.Stderr, "error creating tx", err)
				continue
			}

			for trackingNumber, trackingStatus := range trackingStatuses {
				_, err := tx.Exec(
					"update tracking set status = ? where tracking_number = ?",
					trackingStatus.StatusDescription, trackingNumber,
				)
				if err != nil {
					tx.Rollback()
					fmt.Fprintln(os.Stderr, "error updating tracking number status in DB", err)
					continue
				}
			}

			if err := tx.Commit(); err != nil {
				fmt.Fprintln(os.Stderr, "error commiting tx", err)
				continue
			}

			time.Sleep(60 * time.Second)
		}

	}()

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		dblib.WithGinContext(c, db)
	})

	v1 := r.Group("/api")
	v1.POST("/register", api.Register)
	v1.POST("/login", api.Login)
	v1.POST("/start_tracking", api.StartTrackingGroups)
	v1.POST("/get_tracking", api.GetTrackingNumbers)
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	fmt.Println("Starting server!")
	r.Run(":8080")
}
