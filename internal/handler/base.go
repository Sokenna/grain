package handler

import (
	"github.com/gin-gonic/gin"
	"grain/internal/model"
	"net/http"
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, model.Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应带消息
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, model.Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, model.Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithData 错误响应带数据
func ErrorWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, model.Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
