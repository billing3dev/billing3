package utils

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func JWTSign(claims jwt.MapClaims, duration time.Duration) string {
	claims["nbf"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(duration).Unix()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(jwtKey)
	if err != nil {
		slog.Error("jwt signing failed", "err", err)
		panic(err)
	}
	return signed
}

func JWTVerify(token string) (jwt.MapClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	parsed, err := parser.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("invalid method: %s", t.Method.Alg())
		}
		return jwtKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("jwt verify: %w", err)
	}

	if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("not mapclaims: %v", parsed.Claims)
}

var jwtKey []byte

// read jwt secret from env variables
func InitJWT() {
	jwtKeyString := os.Getenv("JWT_KEY")
	if jwtKeyString == "" {
		slog.Error("JWT_KEY environment not set")
		panic("JWT_KEY environment not set")
	}

	var err error
	jwtKey, err = hex.DecodeString(jwtKeyString)
	if err != nil {
		slog.Error("invalid JWT_KEY", "err", err, "value", jwtKeyString)
		panic("invalid JWT_KEY: " + jwtKeyString)
	}

}
