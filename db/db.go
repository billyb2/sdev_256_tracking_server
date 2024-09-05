package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
)

const dbContextKey = "dbContextKey"

func NewDBConnection() (*sql.DB, error) {
	// default to ./data.db if the DATABASE_PATH env var isn't set
	databasePath := "./data.db"
	if path := os.Getenv("DATABASE_PATH"); path != "" {
		databasePath = path
	}

	// frist, run database migrations if any
	dbmateURL, _ := url.Parse("sqlite:" + databasePath)
	dbmateDB := dbmate.New(dbmateURL)

	if err := dbmateDB.CreateAndMigrate(); err != nil {
		return nil, fmt.Errorf("error migrating: %w", err)
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't open database connection due to error: %w", err)
	}

	return db, nil
}

func WithGinContext(c *gin.Context, db *sql.DB) {
	c.Set(dbContextKey, db)
	c.Next()
}

func FromGinContext(c *gin.Context) *sql.DB {
	db, exists := c.Get(dbContextKey)
	switch exists {
	case true:
		return db.(*sql.DB)
	default:
		return nil
	}
}
