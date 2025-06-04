package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"recharge-go/internal/repository"
	"recharge-go/pkg/logger"
	"recharge-go/pkg/signature"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var platformRepo repository.PlatformRepository

// InitMF178Auth 初始化MF178认证中间件
func InitMF178Auth(db *gorm.DB) {
	platformRepo = repository.NewPlatformRepository(db)
}

// MF178Auth MF178认证中间件
func MF178Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			fmt.Printf("读取请求体失败: %v\n", err)
			response := gin.H{
				"code":    "FAIL",
				"message": "读取请求体失败",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}
		// 恢复请求体，因为后续还需要使用
		c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		fmt.Printf("收到请求体: %s\n", string(body))

		// 解析请求体
		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			logger.Error("[MF178Auth] 解析请求体失败", "error", err, "body", string(body))
			response := gin.H{
				"code":    "FAIL",
				"message": "参数错误",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}

		// 从请求体中获取签名
		sign, ok := req["sign"].(string)
		if !ok || sign == "" {
			fmt.Printf("签名不能为空, 请求体: %+v\n", req)
			response := gin.H{
				"code":    "FAIL",
				"message": "签名不能为空",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}
		// 获取时间戳
		timestamp, ok := req["timestamp"].(float64)
		if !ok {
			fmt.Printf("时间戳不能为空, 请求体: %+v\n", req)
			response := gin.H{
				"code":    "FAIL",
				"message": "时间戳不能为空",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}

		// 验证签名
		userID := c.Param("userid")
		if userID == "" {
			userID = c.Query("userid")
		}
		if userID == "" {
			fmt.Printf("userid 不能为空, 请求体: %+v\n", req)
			response := gin.H{
				"code":    "FAIL",
				"message": "userid 不能为空",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}
		if !signature.VerifySign(req, sign, getAppSecret(userID)) {
			fmt.Printf("签名验证失败, 请求体: %+v\n", req)
			response := gin.H{
				"code":    "FAIL",
				"message": "签名验证失败",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}

		// 验证时间戳
		if !signature.VerifyTimestamp(timestamp, 300) { // 5分钟有效期
			fmt.Printf("验证签名失败: 时间戳过期, timestamp: %v, now: %v, diff: %v秒\n",
				timestamp, time.Now().Unix(), math.Abs(float64(time.Now().Unix())-timestamp))
			response := gin.H{
				"code":    "FAIL",
				"message": "签名验证失败: 时间戳过期",
				"data":    gin.H{},
			}
			c.JSON(http.StatusOK, response)
			c.Abort()
			return
		}

		c.Next()
	}
}

// getAppSecret 获取 app_secret
func getAppSecret(appKey string) string {
	// 从数据库中获取 app_secret
	account, err := platformRepo.GetPlatformAccountByAccountName(appKey)
	if err != nil {
		fmt.Printf("获取平台账号信息失败: %v\n", err)
		return ""
	}
	return account.AppSecret
}
