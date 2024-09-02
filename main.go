package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/billyb2/tracking_server/api"
	dblib "github.com/billyb2/tracking_server/db"
)

func main() {
	ctx := context.Background()
	db, err := dblib.NewDBConnection()
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx = dblib.WithContext(ctx, db)
	defer db.Close()

	fmt.Println("Starting server!")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", api.Register)

	http.ListenAndServe(":8080", func() http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Embed the global context into the request's context
			mux.ServeHTTP(w, r.WithContext(ctx))
		})
	}())
}
