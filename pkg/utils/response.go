package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/pkg/logger"
	"go.uber.org/zap"
	"net/http"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 定义状态码
const (
	CodeSuccess       = 200 // 成功
	CodeInvalidParams = 400 // 参数错误
	CodeUnauthorized  = 401 // 未授权
	CodeForbidden     = 403 // 禁止访问
	CodeNotFound      = 404 // 资源不存在
	CodeInternalError = 500 // 服务器内部错误
)

// 对应的消息
var codeMsgMap = map[int]string{
	CodeSuccess:       "操作成功",
	CodeInvalidParams: "参数错误",
	CodeUnauthorized:  "未授权",
	CodeForbidden:     "禁止访问",
	CodeNotFound:      "资源不存在",
	CodeInternalError: "服务器内部错误",
}

// ResponseWithJSON 返回JSON响应
func ResponseWithJSON(c *gin.Context, code int, data interface{}) {
	msg, ok := codeMsgMap[code]
	if !ok {
		msg = "未知错误"
	}

	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg,
		Data:    data,
	})
}

// ResponseWithData 返回成功响应，包含数据
func ResponseWithData(c *gin.Context, data interface{}) {
	ResponseWithJSON(c, CodeSuccess, data)
}

// ResponseSuccess 返回成功响应，不包含数据
func ResponseSuccess(c *gin.Context) {
	ResponseWithJSON(c, CodeSuccess, nil)
}

// ResponseWithMsg 返回带自定义消息的成功响应
func ResponseWithMsg(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
	})
}

// ResponseWithMsgAndData 返回带自定义消息和数据的成功响应
func ResponseWithMsgAndData(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// ResponseError 返回错误响应
func ResponseError(c *gin.Context, code int, err error) {
	msg, ok := codeMsgMap[code]
	if !ok {
		msg = "未知错误"
	}

	// 如果提供了错误信息，则使用错误信息
	if err != nil {
		msg = err.Error()
	}

	// 记录错误日志
	logger.Error("API错误响应",
		zap.Int("code", code),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("message", msg))

	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: msg,
	})
}

// ResponseBadRequest 返回参数错误响应
func ResponseBadRequest(c *gin.Context, err error) {
	ResponseError(c, CodeInvalidParams, err)
}

// ResponseUnauthorized 返回未授权响应
func ResponseUnauthorized(c *gin.Context, err error) {
	ResponseError(c, CodeUnauthorized, err)
}

// ResponseForbidden 返回禁止访问响应
func ResponseForbidden(c *gin.Context, err error) {
	ResponseError(c, CodeForbidden, err)
}

// ResponseNotFound 返回资源不存在响应
func ResponseNotFound(c *gin.Context, err error) {
	ResponseError(c, CodeNotFound, err)
}

// ResponseInternalError 返回服务器内部错误响应
func ResponseInternalError(c *gin.Context, err error) {
	ResponseError(c, CodeInternalError, err)
}
