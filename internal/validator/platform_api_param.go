package validator

import (
	"recharge-go/internal/model"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidatePlatformAPIParam 验证平台接口参数
func ValidatePlatformAPIParam(param *model.PlatformAPIParam) error {
	validate := validator.New()
	err := validate.Struct(param)
	if err != nil {
		// 将错误信息转换为更友好的格式
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Tag() {
			case "required":
				errMsgs = append(errMsgs, err.Field()+"不能为空")
			case "min":
				errMsgs = append(errMsgs, err.Field()+"不能小于"+err.Param())
			case "max":
				errMsgs = append(errMsgs, err.Field()+"不能大于"+err.Param())
			default:
				errMsgs = append(errMsgs, err.Field()+"验证失败")
			}
		}
		return &ValidationError{Message: strings.Join(errMsgs, "; ")}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
