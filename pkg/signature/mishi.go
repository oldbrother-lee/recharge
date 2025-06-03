package signature

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

func GetMD5(input string) string {
	hash := md5.New()
	hash.Write([]byte(input))
	return strings.ToLower(hex.EncodeToString(hash.Sum(nil)))
}
