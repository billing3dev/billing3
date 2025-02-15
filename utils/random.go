package utils

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

func RandomToken(n int) string {
	buf := make([]byte, n)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

// Randint returns a random integer between a and b, inclusive.
func Randint(a, b int) int {
	if a > b {
		a, b = b, a
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(b-a+1)))
	if err != nil {
		panic(err)
	}
	return a + int(n.Int64())
}

func RandomChoose[S ~[]E, E any](l S) (E, int) {
	if len(l) == 0 {
		panic("empty slice")
	}
	n := Randint(0, len(l)-1)
	return l[n], n
}
