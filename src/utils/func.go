package utils

import (
	"math/rand"
)

func RandomString(size int) string {
	if size <= 0 {size = 5}
	alpha := `abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789` //better to define a const
	buf := make([]byte, size)
	alpha_len := len(alpha)
	for i := 0; i < size; i++ {
	buf[i] = alpha[rand.Intn(alpha_len)]
	}
	return string(buf)
	
}