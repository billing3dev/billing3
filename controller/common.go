package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

const PGErrorUniqueViolation = "23505"

const itemPerPage = 20

type D map[string]any

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	type resp struct {
		Error string `json:"error"`
	}
	json.NewEncoder(w).Encode(resp{
		Error: msg,
	})
}

func writeResp(w http.ResponseWriter, code int, msg D) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	msg["ok"] = true
	json.NewEncoder(w).Encode(msg)
}

func decode[T any](r *http.Request) (*T, error) {
	var t T

	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("content type not application/json")
	}

	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		slog.Debug("decode json", "err", err)
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(&t)
	if err != nil {
		sb := strings.Builder{}
		sb.WriteString("Invalid input: ")
		for _, err := range err.(validator.ValidationErrors) {
			if err.Tag() == "required" {
				sb.WriteString(err.Field())
				sb.WriteString(" is required")
			} else if err.Tag() == "oneof" {
				sb.WriteString(err.Field())
				sb.WriteString(" is invalid")
			} else {
				sb.WriteString(err.Field())
				sb.WriteString(" failed validation: ")
				sb.WriteString(err.Tag())
			}
			sb.WriteString("; ")
		}
		return nil, fmt.Errorf("%s", sb.String())
	}

	return &t, nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	err := tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.Error("rollback tx", "err", err)
	}
}
