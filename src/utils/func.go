package utils

import (
	"math/rand"
	"fmt"
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

func ToUnicode(str string) (result string) {
	result = ""
	for _,c := range str {
		result += fmt.Sprintf("\\x%X", c)
	}
	return 
}