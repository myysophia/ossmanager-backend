package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/utils"
)

// BaseHandler 基础处理器
type BaseHandler struct {
	// 可以添加一些共享的依赖，比如数据库连接、服务实例等
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// Success 成功响应
func (h *BaseHandler) Success(c *gin.Context, data interface{}) {
	utils.Success(c, data)
}

// Error 错误响应
func (h *BaseHandler) Error(c *gin.Context, code int, message string) {
	utils.Error(c, code, message)
}

// BadRequest 请求参数错误
func (h *BaseHandler) BadRequest(c *gin.Context, message string) {
	utils.Error(c, utils.CodeBadRequest, message)
}

// Unauthorized 未授权
func (h *BaseHandler) Unauthorized(c *gin.Context, message string) {
	utils.Error(c, utils.CodeUnauthorized, message)
}

// Forbidden 禁止访问
func (h *BaseHandler) Forbidden(c *gin.Context, message string) {
	utils.Error(c, utils.CodeForbidden, message)
}

// NotFound 资源不存在
func (h *BaseHandler) NotFound(c *gin.Context, message string) {
	utils.Error(c, utils.CodeNotFound, message)
}

// InternalError 内部错误
func (h *BaseHandler) InternalError(c *gin.Context, message string) {
	utils.Error(c, utils.CodeInternalError, message)
}
