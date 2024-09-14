package auth

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"

	"github.com/billyb2/tracking_server/db"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/argon2"
)

var BadUsernameOrPassword error = fmt.Errorf("incorrect username or password")

func CheckUserCreds(c *gin.Context, username, password string) (string, error) {
	db := db.FromGinContext(c)
	if db == nil {
		return "", fmt.Errorf("db is nil")
	}

	row := db.QueryRow("select id, password_hash, salt from users where username = ?", username)
	if row.Err() != nil {
		return "", row.Err()
	}

	var userID int32
	var passwordHash []byte
	var passwordSalt []byte
	if err := row.Scan(&userID, &passwordHash, &passwordSalt); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", BadUsernameOrPassword
		default:
			return "", err
		}
	}

	passwordBytes := []byte(password)
	attemptedPasswordHash := argon2.IDKey(passwordBytes, passwordSalt, 1, 47104, 1, 32)
	if !bytes.Equal(passwordHash, attemptedPasswordHash) {
		return "", BadUsernameOrPassword
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Commit()
	token, err := CreateToken(tx, userID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return token, nil
}

func CreateToken(tx *sql.Tx, userID int32) (string, error) {
	token := ulid.Make()
	_, err := tx.Exec("insert into tokens (user_id, token) values (?, ?)", userID, token.String())
	if err != nil {
		return "", err
	}

	return token.String(), nil
}

var InvalidToken error = fmt.Errorf("invalid token")

func UserIDFromToken(c *gin.Context, token string) (int32, error) {
	db := db.FromGinContext(c)
	if db == nil {
		return 0, fmt.Errorf("db is nil")
	}
	row := db.QueryRow("select user_id from tokens where token = ?", token)
	if row == nil {
		return 0, InvalidToken
	}

	var userID int32
	if err := row.Scan(&userID); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, InvalidToken
		default:
			return 0, err
		}
	}

	return userID, nil
}
