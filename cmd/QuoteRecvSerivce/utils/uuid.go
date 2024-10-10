package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func randomHex(n int) string {
	rand.Seed(time.Now().UnixNano())
	hexChars := "0123456789abcdef"
	bytes := make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = hexChars[rand.Intn(len(hexChars))]
	}
	return string(bytes)
}

func getUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", randomHex(8), randomHex(4), randomHex(4), randomHex(4), randomHex(12))
}
