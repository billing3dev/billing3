package controller

import (
	"billing3/database"
	"billing3/utils"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func adminUserList(w http.ResponseWriter, r *http.Request) {
	count, err := database.Q.SearchUsersCount(r.Context(), r.URL.Query().Get("search"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin user table", "err", err)
		return
	}

	if count == 0 {
		writeResp(w, http.StatusOK, D{
			"users":       make([]any, 0),
			"total_pages": 0,
		})
		return
	}

	totalPages := int(math.Ceil(float64(count) / itemPerPage))

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	users, err := database.Q.SearchUsersPaged(r.Context(), database.SearchUsersPagedParams{
		Search: r.URL.Query().Get("search"),
		Limit:  itemPerPage,
		Offset: int32(itemPerPage * (page - 1)),
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin user table", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{
		"users":       users,
		"total_pages": totalPages,
	})
}

func adminUserEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Email    string `json:"email" validate:"required"`
		Name     string `json:"name" validate:"required"`
		Password string `json:"password" validate:"printascii,max=72"`
		Role     string `json:"role" validate:"required"`
		Address  string `json:"address"`
		City     string `json:"city"`
		State    string `json:"state"`
		Country  string `json:"country"`
		ZipCode  string `json:"zip_code"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// update password if provided
	if req.Password != "" {
		err = database.Q.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
			ID:       int32(id),
			Password: utils.HashPassword(req.Password),
		})
		if err != nil {
			slog.Error("admin user edit", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = database.Q.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:      int32(id),
		Email:   req.Email,
		Name:    req.Name,
		Role:    req.Role,
		Address: pgtype.Text{Valid: req.Address != "", String: req.Address},
		City:    pgtype.Text{Valid: req.City != "", String: req.City},
		State:   pgtype.Text{Valid: req.State != "", String: req.State},
		Country: pgtype.Text{Valid: req.Country != "", String: req.Country},
		ZipCode: pgtype.Text{Valid: req.ZipCode != "", String: req.ZipCode},
	})
	if err != nil {
		slog.Error("admin user edit", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminUserGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	user, err := database.Q.FindUserById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("admin user get", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{
		"user": user,
	})
}

func adminUserCreate(w http.ResponseWriter, r *http.Request) {
	type reqStruct struct {
		Email    string `json:"email" validate:"required"`
		Name     string `json:"name" validate:"required"`
		Password string `json:"password" validate:"required,printascii,max=72"`
		Role     string `json:"role" validate:"required"`
		Address  string `json:"address"`
		City     string `json:"city"`
		State    string `json:"state"`
		Country  string `json:"country"`
		ZipCode  string `json:"zip_code"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := database.Q.CreateUser(r.Context(), database.CreateUserParams{
		Email:    req.Email,
		Name:     req.Name,
		Password: utils.HashPassword(req.Password),
		Role:     req.Role,
		Address:  pgtype.Text{Valid: req.Address != "", String: req.Address},
		City:     pgtype.Text{Valid: req.City != "", String: req.City},
		State:    pgtype.Text{Valid: req.State != "", String: req.State},
		Country:  pgtype.Text{Valid: req.Country != "", String: req.Country},
		ZipCode:  pgtype.Text{Valid: req.ZipCode != "", String: req.ZipCode},
	})
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
			writeError(w, http.StatusForbidden, "Duplicated email")
			return
		}
		slog.Error("admin create user", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{
		"id": id,
	})
}
