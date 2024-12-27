package controller

import (
	"billing3/controller/middlewares"
	"net/http"
)

// returns the authenticated user
func me(w http.ResponseWriter, r *http.Request) {
	user := middlewares.MustGetUser(r)

	writeResp(w, http.StatusOK, D{
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
	})
}
