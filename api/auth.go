package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/billyb2/tracking_server/db"
	"golang.org/x/crypto/argon2"
)

type RegistrationInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Company  string `json:"company"`
}

type registerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid method, expected POST"))
		return
	}

	defer r.Body.Close()
	jsonDecoder := json.NewDecoder(r.Body)

	authInfo := RegistrationInfo{}
	err := jsonDecoder.Decode(&authInfo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("error parsing RegistrationInfo: %s", err)))
		return
	}

	statusCode := http.StatusOK
	resp := registerResponse{
		Success: true,
	}

	if err := registerUser(r.Context(), &authInfo); err != nil {
		switch {
		case errors.Is(err, &duplicateUserError{}):
			statusCode = http.StatusBadRequest
		default:
			statusCode = http.StatusInternalServerError
		}
		resp.Success = false
		resp.Error = err.Error()
	}

	respBin, err := json.Marshal(&resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}

	w.WriteHeader(statusCode)
	w.Write(respBin)
	return

}

type duplicateUserError struct{}

func (e *duplicateUserError) Error() string {
	return "a user with that username already exists"
}

func registerUser(ctx context.Context, authInfo *RegistrationInfo) error {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	if err != nil {
		return err
	}

	passwordHash := argon2.IDKey([]byte(authInfo.Password), salt, 1, 47104, 1, 32)

	db := db.FromContext(ctx)
	_, err = db.ExecContext(ctx, "insert into users (username, password_hash, salt, company) values (?, ?, ?, ?)", authInfo.Username, passwordHash, salt, authInfo.Company)
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
