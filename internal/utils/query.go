package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetIntQuery 获取整数类型的查询参数
func GetIntQuery(ctx *gin.Context, key string, defaultValue int) int {
	value := ctx.DefaultQuery(key, strconv.Itoa(defaultValue))
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetInt64Query 获取int64类型的查询参数
func GetInt64Query(ctx *gin.Context, key string, defaultValue int64) int64 {
	value := ctx.DefaultQuery(key, strconv.FormatInt(defaultValue, 10))
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return result
}
