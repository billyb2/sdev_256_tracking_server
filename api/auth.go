package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/billyb2/tracking_server/db"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/argon2"
)

type RegistrationInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Company  string `json:"company"`
}

type registerResponse struct {
	Token string `json:"token"`
	Error string `json:"error"`
}

// Register godoc
//
//	@Summary	Registers a new user
//	@ID			register-user
//	@Accept		json
//	@Produce	json
//	@Param		registrationInfo	body		RegistrationInfo	true	"Registration Info"
//	@Success	201					{object}	registerResponse
//	@Failure	400					{object}	registerResponse
//	@Failure	500					{object}	registerResponse
//	@Router		/register [post]
func Register(c *gin.Context) {
	authInfo := RegistrationInfo{}
	err := c.BindJSON(&authInfo)
	if err != nil {
		err = fmt.Errorf("error parsing RegistrationInfo: %w")
		c.JSON(http.StatusBadRequest, registerResponse{
			Error: err.Error(),
			Token: "",
		})
		return
	}

	statusCode := http.StatusCreated
	resp := registerResponse{
		Token: "abc123",
	}

	if err := registerUser(c, &authInfo); err != nil {
		switch {
		case errors.Is(err, &duplicateUserError{}):
			statusCode = http.StatusBadRequest
		default:
			statusCode = http.StatusInternalServerError
		}
		resp.Error = err.Error()
	}

	c.JSON(statusCode, resp)
	return
}

type duplicateUserError struct{}

func (e *duplicateUserError) Error() string {
	return "a user with that username already exists"
}

func registerUser(c *gin.Context, authInfo *RegistrationInfo) error {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return err
	}

	passwordHash := argon2.IDKey([]byte(authInfo.Password), salt, 1, 47104, 1, 32)

	db := db.FromGinContext(c)
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	_, err = db.Exec("insert into users (username, password_hash, salt, company) values (?, ?, ?, ?)", authInfo.Username, passwordHash, salt, authInfo.Company)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "UNIQUE constraint failed: users.username"):
			return &duplicateUserError{}
		default:
			return err
		}
	}

	return nil
}
