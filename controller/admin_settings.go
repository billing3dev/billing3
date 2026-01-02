package controller

import (
	"billing3/service"
	"net/http"
)

func adminSettingsList(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]string)

	for _, s := range service.Settings {
		m[s.Key()] = s.Get(r.Context())
	}

	writeResp(w, http.StatusOK, D{"data": m})
}

func adminSettingsUpdate(w http.ResponseWriter, r *http.Request) {

	req, err := decode[map[string]string](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	for _, s := range service.Settings {
		if v, ok := (*req)[s.Key()]; ok {
			s.Set(r.Context(), v)
		}
	}

	writeResp(w, http.StatusOK, D{})
}
