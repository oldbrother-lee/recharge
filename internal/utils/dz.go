package utils

// DzOperatorIDToCode 将得众平台的 operator_id 映射为内部 ISP 编码
// 约定：1=移动 2=电信 3=联通；其他返回0（未知）
func DzOperatorIDToCode(opID int) int {
	switch opID {
	case 1:
		return 1 // 移动
	case 2:
		return 2 // 电信
	case 3:
		return 3 // 联通
	default:
		return 0
	}
}