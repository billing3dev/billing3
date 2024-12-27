package controller

import (
	"billing3/database"
	"billing3/service/extension"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
)

func adminServerList(w http.ResponseWriter, r *http.Request) {
	servers, err := database.Q.ListServers(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin server list", "err", err)
		return
	}

	for _, server := range servers {
		server.Settings = make(map[string]string)
	}

	writeResp(w, http.StatusOK, D{"servers": servers})
}

func adminServerGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	server, err := database.Q.FindServerById(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin server get", "err", err)
		return
	}

	writeResp(w, http.StatusOK, D{"server": server})
}

type adminServerReq struct {
	Extension string            `json:"extension" validate:"required"`
	Label     string            `json:"label" validate:"required"`
	Settings  map[string]string `json:"settings"`
}

func adminServerValidate(req *adminServerReq) (map[string]string, error) {
	ext, ok := extension.Extensions[req.Extension]
	if !ok {
		return nil, fmt.Errorf("extension \"%s\" not found", req.Extension)
	}

	cleanedSettings := make(map[string]string)

	for _, setting := range ext.ServerSettings() {
		userInput, ok := req.Settings[setting.Name]
		if !ok {
			userInput = ""
		}

		if setting.Type == "select" {
			if slices.Contains(setting.Values, userInput) {
				return nil, fmt.Errorf("%s is invalid", setting.DisplayName)
			}
		} else {
			if setting.Regex != "" {
				compile, err := regexp.Compile(setting.Regex)
				if err != nil {
					slog.Error("invalid regex", "err", err, "regex", setting.Regex, "extension", req.Extension, "setting", setting.Name)
					return nil, fmt.Errorf("internal error")
				}
				if !compile.MatchString(userInput) {
					return nil, fmt.Errorf("%s is invalid", setting.DisplayName)
				}
			}
		}

		cleanedSettings[setting.Name] = userInput
	}

	return cleanedSettings, nil
}

func adminServerAdd(w http.ResponseWriter, r *http.Request) {
	req, err := decode[adminServerReq](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cleanedSettings, err := adminServerValidate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := database.Q.CreateServer(r.Context(), database.CreateServerParams{
		Label:     req.Label,
		Extension: req.Extension,
		Settings:  cleanedSettings,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("admin server add", "err", err)
		return
	}

	writeResp(w, http.StatusCreated, D{"server": id})
}

func adminServerEdit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req, err := decode[adminServerReq](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cleanedSettings, err := adminServerValidate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = database.Q.UpdateServer(r.Context(), database.UpdateServerParams{
		Label:    req.Label,
		Settings: cleanedSettings,
		ID:       int32(id),
	})
	if err != nil {
		slog.Error("admin server edit", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminServerDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	count, err := database.Q.CountServicesByServer(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin server delete", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if count > 0 {
		writeError(w, http.StatusBadRequest, "server is currently in use")
		return
	}

	err = database.Q.DeleteServer(r.Context(), int32(id))
	if err != nil {
		slog.Error("admin server delete", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeResp(w, http.StatusOK, D{})
}

func adminExtensionServerSettings(w http.ResponseWriter, r *http.Request) {
	extensionName := r.URL.Query().Get("extension")

	ext, ok := extension.Extensions[extensionName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeResp(w, http.StatusOK, D{"settings": ext.ServerSettings()})
}
