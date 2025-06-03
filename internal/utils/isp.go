package utils

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
