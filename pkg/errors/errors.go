package errors

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorCode 错误码类型
type ErrorCode int

const (
	// 通用错误码
	Success         ErrorCode = 0
	InternalError   ErrorCode = 1000
	InvalidParams   ErrorCode = 1001
	Unauthorized    ErrorCode = 1002
	Forbidden       ErrorCode = 1003
	NotFound        ErrorCode = 1004
	Conflict        ErrorCode = 1005
	TooManyRequests ErrorCode = 1006

	// 用户相关错误码
	UserNotFound    ErrorCode = 2001
	UserExists      ErrorCode = 2002
	InvalidPassword ErrorCode = 2003
	UserDisabled    ErrorCode = 2004

	// 订单相关错误码
	OrderNotFound     ErrorCode = 3001
	OrderExists       ErrorCode = 3002
	OrderStatusError  ErrorCode = 3003
	InsufficientFunds ErrorCode = 3004

	// 充值相关错误码
	RechargeNotFound ErrorCode = 4001
	RechargeFailed   ErrorCode = 4002
	RechargeTimeout  ErrorCode = 4003

	// 平台相关错误码
	PlatformNotFound ErrorCode = 5001
	PlatformError    ErrorCode = 5002
	PlatformTimeout  ErrorCode = 5003
)

// AppError 应用错误结构
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Err     error     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Response 错误响应结构
type Response struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Details string      `json:"details,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) Response {
	return Response{
		Code:    Success,
		Message: "success",
		Data:    data,
	}
}

// ErrorResponse 错误响应
func ErrorResponse(err *AppError) Response {
	return Response{
		Code:    err.Code,
		Message: err.Message,
		Details: err.Details,
	}
}

// HandleError 统一错误处理中间件
func HandleError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		statusCode := getHTTPStatusCode(appErr.Code)
		c.JSON(statusCode, ErrorResponse(appErr))
	} else {
		// 未知错误，返回内部服务器错误
		appErr := Wrap(err, InternalError, "Internal server error")
		c.JSON(http.StatusInternalServerError, ErrorResponse(appErr))
	}
	c.Abort()
}

// getHTTPStatusCode 根据错误码获取HTTP状态码
func getHTTPStatusCode(code ErrorCode) int {
	switch code {
	case Success:
		return http.StatusOK
	case InvalidParams:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound, UserNotFound, OrderNotFound, RechargeNotFound, PlatformNotFound:
		return http.StatusNotFound
	case Conflict, UserExists, OrderExists:
		return http.StatusConflict
	case TooManyRequests:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// 预定义常用错误
var (
	ErrInternalServer  = New(InternalError, "Internal server error")
	ErrInvalidParams   = New(InvalidParams, "Invalid parameters")
	ErrUnauthorized    = New(Unauthorized, "Unauthorized")
	ErrForbidden       = New(Forbidden, "Forbidden")
	ErrNotFound        = New(NotFound, "Resource not found")
	ErrUserNotFound    = New(UserNotFound, "User not found")
	ErrUserExists      = New(UserExists, "User already exists")
	ErrInvalidPassword = New(InvalidPassword, "Invalid password")
	ErrOrderNotFound   = New(OrderNotFound, "Order not found")
	ErrRechargeFailed  = New(RechargeFailed, "Recharge failed")
)
