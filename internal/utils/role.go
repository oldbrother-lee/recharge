package utils

// HasRole 检查用户是否具有指定角色
// roles: 用户角色列表
// targetRole: 目标角色
// 返回值: 如果用户具有目标角色返回 true，否则返回 false
func HasRole(roles []string, targetRole string) bool {
	for _, role := range roles {
		if role == targetRole {
			return true
		}
	}
	return false
}
