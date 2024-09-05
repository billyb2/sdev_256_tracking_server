package api

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/billyb2/tracking_server/db"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/argon2"
)

type registrationInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Company  string `json:"company"`
}

type registerResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

// Register godoc
//
//	@Summary	Registers a new user
//	@ID			register-user
//	@Accept		json
//	@Produce	json
//	@Param		registrationInfo	body		registrationInfo	true	"Registration Info"
//	@Success	200					{object}	registerResponse
//	@Failure	401					{object}	registerResponse
//	@Failure	500					{object}	registerResponse
//	@Router		/register [post]
func Register(c *gin.Context) {
	authInfo := registrationInfo{}
	err := c.BindJSON(&authInfo)
	if err != nil {
		err = fmt.Errorf("error parsing RegistrationInfo: %w")
		c.JSON(http.StatusBadRequest, registerResponse{
			Error: err.Error(),
		})
		return
	}

	statusCode := http.StatusCreated
	resp := registerResponse{}

	token, err := registerUser(c, &authInfo)

	switch {
	case errors.Is(err, duplicateUserError):
		statusCode = http.StatusBadRequest
		resp.Error = err.Error()
	case err != nil:
		statusCode = http.StatusInternalServerError
		resp.Error = err.Error()
	default:
		resp.Token = token
	}

	c.JSON(statusCode, resp)
	return
}

var duplicateUserError error = fmt.Errorf("a user with that username already exists")

func registerUser(c *gin.Context, authInfo *registrationInfo) (string, error) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	passwordHash := argon2.IDKey([]byte(authInfo.Password), salt, 1, 47104, 1, 32)

	db := db.FromGinContext(c)
	if db == nil {
		return "", fmt.Errorf("db is nil")
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Commit()

	row := tx.QueryRow("insert into users (username, password_hash, salt, company) values (?, ?, ?, ?) returning id", authInfo.Username, passwordHash, salt, authInfo.Company)
	if row.Err() != nil {
		return "", err
	}
	var userID int32
	if err := row.Scan(&userID); err != nil {
		switch {
		case strings.Contains(err.Error(), "UNIQUE constraint failed: users.username"):
			return "", duplicateUserError
		default:
			return "", err
		}
	}

	token, err := createToken(tx, userID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err

	}

	return token, err

}

func createToken(tx *sql.Tx, userID int32) (string, error) {
	token := ulid.Make()
	_, err := tx.Exec("insert into tokens (user_id, token) values (?, ?)", userID, token)
	if err != nil {
		return "", err
	}

	return token.String(), nil

}

type loginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

// Login godoc
//
//	@Summary	Verifies and logs in the user, returning a token
//	@ID			login-user
//	@Accept		json
//	@Produce	json
//	@Param		loginInfo	body		loginInfo	true	"Login Info"
//	@Success	200			{object}	loginResponse
//	@Failure	401			{object}	loginResponse
//	@Failure	500			{object}	loginResponse
//	@Router		/login [post]
func Login(c *gin.Context) {
	authInfo := loginInfo{}
	err := c.BindJSON(&authInfo)
	if err != nil {
		err = fmt.Errorf("error parsing loginResponse: %w")
		c.JSON(http.StatusBadRequest, loginResponse{
			Error: err.Error(),
		})
		return
	}

	token, err := checkUserCreds(c, &authInfo)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, loginResponse{
			Token: token,
		})

	case errors.Is(err, badUsernameOrPassword):
		c.JSON(http.StatusUnauthorized, loginResponse{
			Error: err.Error(),
		})

	default:
		c.JSON(http.StatusInternalServerError, loginResponse{
			Error: err.Error(),
		})
	}
}

var badUsernameOrPassword error = fmt.Errorf("incorrect username or password")

func checkUserCreds(c *gin.Context, authInfo *loginInfo) (string, error) {
	db := db.FromGinContext(c)
	if db == nil {
		return "", fmt.Errorf("db is nil")
	}

	row := db.QueryRow("select id, password_hash, salt from users where username = ?", authInfo.Username)
	if row.Err() != nil {
		return "", row.Err()
	}

	var userID int32
	var passwordHash []byte
	var passwordSalt []byte
	if err := row.Scan(&userID, &passwordHash, &passwordSalt); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", badUsernameOrPassword
		default:
			return "", err
		}
	}

	passwordBytes := []byte(authInfo.Password)
	attemptedPasswordHash := argon2.IDKey(passwordBytes, passwordSalt, 1, 47104, 1, 32)
	if !bytes.Equal(passwordHash, attemptedPasswordHash) {
		return "", badUsernameOrPassword
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Commit()
	token, err := createToken(tx, userID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return token, nil
}
