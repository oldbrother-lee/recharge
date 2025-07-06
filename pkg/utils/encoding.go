package utils

import (
	"io/ioutil"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// EncodeGBK 将字符串转为GBK编码
func EncodeGBK(s string) ([]byte, error) {
	reader := transform.NewReader(strings.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	return ioutil.ReadAll(reader)
}

// DecodeGBK 将GBK编码转为UTF-8字符串
func DecodeGBK(data []byte) (string, error) {
	reader := transform.NewReader(strings.NewReader(string(data)), simplifiedchinese.GBK.NewDecoder())
	utf8Data, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(utf8Data), nil
}