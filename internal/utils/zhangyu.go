package utils

// ZhangyuOperatorIDToCode 将章鱼平台的 operator_id 映射为内部 ISP 编码
// 章鱼约定：1=移动 2=联通 3=电信；系统内部约定：1=移动 2=电信 3=联通
// 因此需要在 2 与 3 间进行互换
func ZhangyuOperatorIDToCode(opID int) int {
    switch opID {
    case 1:
        return 1 // 移动
    case 2:
        return 3 // 章鱼联通 -> 系统联通(3)
    case 3:
        return 2 // 章鱼电信 -> 系统电信(2)
    default:
        return 0 // 未知
    }
}