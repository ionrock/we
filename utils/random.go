package utils

import (
	"math/rand"
	"time"
)

var letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenRandEnvVar() (string, string) {
	return RandString(7), RandString(5)
}
