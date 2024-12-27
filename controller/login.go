package controller

import (
	"billing3/database"
	"billing3/service"
	"billing3/utils"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func login(w http.ResponseWriter, r *http.Request) {
	type reqStruct struct {
		Email    string `json:"email" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := database.Q.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// user does not exist
			writeError(w, http.StatusOK, "Wrong email or password")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("login", "err", err)
		return
	}

	if !utils.ComparePassword(user.Password, req.Password) {
		// wrong password
		writeError(w, http.StatusOK, "Wrong email or password")
		return
	}

	token, err := service.NewSessionToken(r.Context(), user.ID)
	if err != nil {
		slog.Error("login", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{
		"token": token,
	})
}
