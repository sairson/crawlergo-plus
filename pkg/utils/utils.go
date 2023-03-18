package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func ConvertHeaders(h map[string]interface{}) map[string]string {
	a := map[string]string{}
	for key, value := range h {
		a[key] = value.(string)
	}
	return a
}

func CalcMD5Hash(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
