package controller

import (
	"billing3/service"
	"net/http"
)

func publicSettings(w http.ResponseWriter, r *http.Request) {
	m := make(map[string]string)

	for _, s := range service.Settings {
		if s.IsPublic() {
			m[s.Key()] = s.Get(r.Context())
		}
	}

	writeResp(w, http.StatusOK, D{"data": m})
}
