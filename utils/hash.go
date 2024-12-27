package utils

import (
	"log/slog"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plaintext string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), 0)
	if err != nil {
		slog.Error("bcrypt", "err", err)
		panic(err)
	}
	return string(hashed)
}

func ComparePassword(hashed, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext)) == nil
}
