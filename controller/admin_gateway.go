package controller

import (
	"billing3/database"
	"billing3/service/gateways"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func adminListGateways(w http.ResponseWriter, r *http.Request) {
	g, err := database.Q.ListGateways(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin list gateways", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{"gateways": g})
}

func adminGatewayGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	gateway, err := database.Q.FindGatewayById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("admin get gateway", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{"gateway": gateway})
}

func adminGatewayUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type reqStruct struct {
		DisplayName string            `json:"display_name" validate:"required"`
		Settings    map[string]string `json:"settings"`
		Enabled     bool              `json:"enabled"`
		Fee         string            `json:"fee" validate:"required"`
	}
	req, err := decode[reqStruct](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	gateway, err := database.Q.FindGatewayById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		slog.Error("admin get gateway", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gatewayObject, ok := gateways.Gateways[gateway.Name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// validation

	// validate settings
	cleanedSettings := make(map[string]string)
	for _, setting := range gatewayObject.Settings() {
		userInput, ok := req.Settings[setting.Name]
		if !ok {
			userInput = ""
		}

		switch setting.Type {

		case "select":
			if !slices.Contains(setting.Values, userInput) {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("\"%s\" must be one of: %s", setting.DisplayName, strings.Join(setting.Values, ", ")))
				return
			}

		case "string":
			fallthrough
		case "text":
			if setting.Regex != "" {
				compile, err := regexp.Compile(setting.Regex)
				if err != nil {
					slog.Error("invalid gateway setting regex", "gateway", gateway.Name, "name", setting.Name, "regex", setting.Regex, "err", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				if !compile.MatchString(userInput) {
					writeError(w, http.StatusBadGateway, "\""+setting.DisplayName+"\" is invalid")
					return
				}
			}

		default:
			w.WriteHeader(http.StatusInternalServerError)
			slog.Error("invalid gateway setting type", "gateway", gateway.Name, "name", setting.Name, "type", setting.Type)
			return
		}

		cleanedSettings[setting.Name] = userInput
	}

	// validate fee

	// Fee may be a number representing a fixed value,
	_, err = decimal.NewFromString(strings.TrimSuffix(req.Fee, "%"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "fee is invalid")
		return
	}

	err = database.Q.UpdateGateway(r.Context(), database.UpdateGatewayParams{
		DisplayName: req.DisplayName,
		Settings:    cleanedSettings,
		Enabled:     req.Enabled,
		Fee:         pgtype.Text{Valid: true, String: req.Fee},
		Name:        gateway.Name,
	})
	if err != nil {
		if err, ok := err.(*pgconn.PgError); ok && err.Code == PGErrorUniqueViolation {
			writeError(w, http.StatusForbidden, "duplicated display name")
			return
		}
		slog.Error("admin update gateway", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func adminGatewaySettings(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	gateway, ok := gateways.Gateways[name]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeResp(w, http.StatusOK, D{"settings": gateway.Settings()})
}
