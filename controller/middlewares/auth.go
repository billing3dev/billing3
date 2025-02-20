package middlewares

import (
	"billing3/database"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type authCtx string

// Auth adds user to context if session token is valid
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")

		ctx := r.Context()

		session, err := database.Q.FindSessionByToken(ctx, token)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// continue if session token is invalid
				next.ServeHTTP(w, r)
				return
			}
			slog.Error("find session", "err", err, "token", token)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		user, err := database.Q.FindUserById(ctx, session.UserID)
		if err != nil {
			slog.Error("find session", "err", err, "token", token)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx = context.WithValue(ctx, authCtx("USER"), &user)
		ctx = context.WithValue(ctx, authCtx("TOKEN"), token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MustAuth blocks any request that does not have a user
func MustAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUser(r) == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, "{\"error\": \"Unauthorized\"}")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireRole(role string) func(next http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r)
			if user == nil || user.Role != role {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				io.WriteString(w, "{\"error\": \"unauthorized\"}")
				return
			}
			next.ServeHTTP(w, r)
		})
	}

}

// MustGetUser panics if user does not exist
func MustGetUser(r *http.Request) *database.User {
	user, ok := r.Context().Value(authCtx("USER")).(*database.User)
	if !ok || user == nil {
		panic("user not in context")
	}
	return user
}

func MustGetToken(r *http.Request) string {
	token, ok := r.Context().Value(authCtx("TOKEN")).(string)
	if !ok {
		panic("token not in context")
	}
	return token
}

// GetUser returns nil if user does not exist
func GetUser(r *http.Request) *database.User {
	user, ok := r.Context().Value(authCtx("USER")).(*database.User)
	if !ok || user == nil {
		return nil
	}
	return user
}
