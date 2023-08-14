package utils

import (
	"math/rand"
	"strings"
)

func SplitNoEmpty(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	res := strings.Split(s, sep)
	for i := 0; i < len(res); i++ {
		if res[i] == "" {
			res = append(res[:i], res[i+1:]...)
		}
	}

	return res
}

func JoinNoEmpty(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}

	return strings.Join(s, sep)
}

func RandomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[RandomInt(0, len(letterBytes))]
	}
	return string(b)
}

func RandomInt(min, max int) int {
	return min + rand.Intn(max-min)
}
