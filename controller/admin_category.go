package controller

import (
	"billing3/database"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func adminCategoryList(w http.ResponseWriter, r *http.Request) {
	categories, err := database.Q.ListCategories(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin list category", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{
		"categories": categories,
	})
}

func adminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	type reqStruct struct {
		Name        string `json:"name" validate:"required,max=200"`
		Description string `json:"description"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := database.Q.CreateCategory(r.Context(), database.CreateCategoryParams{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		slog.Error("admin create category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{
		"id": id,
	})
}

func adminCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		Name        string `json:"name" validate:"required,max=200"`
		Description string `json:"description"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = database.Q.UpdateCategory(r.Context(), database.UpdateCategoryParams{
		Name:        req.Name,
		Description: req.Description,
		ID:          int32(id),
	})
	if err != nil {
		slog.Error("admin update category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminCategoryGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	c, err := database.Q.FindCategoryById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin category get", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{"category": c})
}

func adminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = database.Q.DeleteCategory(r.Context(), int32(id))
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == "23503" {
			writeError(w, http.StatusForbidden, "the category cannot be deleted if there are products belonging to it")
			return
		}
		slog.Error("admin delete category", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}
