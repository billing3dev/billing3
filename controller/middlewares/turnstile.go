package middlewares

import (
	"billing3/service"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

func CloudflareTurnstile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		turnstileResp := r.Header.Get("X-Turnstile")
		secret := service.SettingTurnstileSecret.Get(r.Context())

		if secret == "" {
			// turnstile not configured, skip verification
			next.ServeHTTP(w, r)
			return
		}

		if turnstileResp == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, "{\"error\": \"Invalid CAPTCHA\"}")
			return
		}

		resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
			"secret":   []string{secret},
			"response": []string{turnstileResp},
		})
		if err != nil {
			slog.Error("turnstile verify", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		type respStruct struct {
			Success    bool     `json:"success"`
			ErrorCodes []string `json:"error-codes"`
		}

		var result respStruct
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			slog.Error("turnstile verify", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !result.Success {
			slog.Debug("turnstile error codes", "error codes", result.ErrorCodes)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, "{\"error\": \"Invalid CAPTCHA\"}")
			return
		}

		next.ServeHTTP(w, r)
	})
}
