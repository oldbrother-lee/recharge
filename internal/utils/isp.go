package utils

import (
	"fmt"
	"regexp"
	"strconv"
)

// ISPNameToCode 将运营商名称转为编码（1:移动 2:电信 3:联通 0:未知）
func ISPNameToCode(name string) int {
	switch name {
	case "移动":
		return 1
	case "电信":
		return 2
	case "联通":
		return 3
	case "中国移动":
		return 1
	case "中国电信":
		return 2
	case "中国联通":
		return 3
	default:
		return 0
	}
}

// ExtractNumberFromProductName 从产品名称中提取数字部分
// 例如："全国联通50" -> 50, "移动100元" -> 100
func ExtractNumberFromProductName(productName string) (float64, error) {
	// 使用正则表达式提取数字
	re := regexp.MustCompile(`\d+(?:\.\d+)?`)
	match := re.FindString(productName)
	if match == "" {
		return 0, fmt.Errorf("未能从产品名称 '%s' 中提取到数字", productName)
	}
	
	value, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0, fmt.Errorf("解析数字失败: %v", err)
	}
	
	return value, nil
}
