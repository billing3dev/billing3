package controller

import (
	"billing3/controller/middlewares"
	"fmt"
	"net/http"
)

// returns the authenticated user
func me(w http.ResponseWriter, r *http.Request) {
	user := middlewares.MustGetUser(r)

	address := fmt.Sprintf("%s, %s, %s, %s, %s", user.Address.String, user.City.String, user.State.String, user.Country.String, user.ZipCode.String)
	if !user.Address.Valid && !user.City.Valid && !user.State.Valid && !user.Country.Valid && !user.ZipCode.Valid {
		address = "unknown"
	}

	writeResp(w, http.StatusOK, D{
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
		"address": address,
	})
}
