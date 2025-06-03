package model

// 运营商常量定义
const (
	OperatorMobile  = 1 // 移动
	OperatorUnicom  = 2 // 电信
	OperatorTelecom = 3 // 联通
)

// 运营商映射表
var OperatorNameMap = map[int]string{
	OperatorMobile:  "移动",
	OperatorUnicom:  "电信",
	OperatorTelecom: "联通",
}
