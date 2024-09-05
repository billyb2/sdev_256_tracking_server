package api

import (
	"crypto/rand"
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
//	@Success	201					{object}	registerResponse
//	@Failure	400					{object}	registerResponse
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

	token := ulid.Make()
	_, err = tx.Exec("insert into tokens (user_id, token) values (?, ?)", userID, token)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}

	return token.String(), nil
}
