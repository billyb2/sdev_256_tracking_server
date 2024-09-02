package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
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

func WithContext(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}

func FromContext(ctx context.Context) *sql.DB {
	return ctx.Value(dbContextKey).(*sql.DB)
}
