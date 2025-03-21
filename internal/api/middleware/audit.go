package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"go.uber.org/zap"
	"strings"
)

// AuditLogMiddleware 审计日志中间件
func AuditLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 处理请求
		c.Next()

		// 仅记录修改操作的审计日志
		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			return
		}

		// 获取用户信息
		userID, exists := c.Get("userID")
		if !exists {
			return
		}
		username, _ := c.Get("username")

		// 构建审计日志
		auditLog := models.AuditLog{
			UserID:       userID.(uint),
			Username:     username.(string),
			Action:       method,
			ResourceType: getResourceType(c.Request.URL.Path),
			ResourceID:   getResourceID(c.Request.URL.Path),
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			Status:       c.Writer.Status(),
		}

		// 异步保存审计日志
		go func(log models.AuditLog) {
			if err := db.GetDB().Create(&log).Error; err != nil {
				logger.Error("保存审计日志失败", zap.Error(err))
			}
		}(auditLog)
	}
}

// getResourceType 从请求路径中获取资源类型
func getResourceType(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return "unknown"
	}

	// 例如 /api/v1/oss/files/1 => oss_file
	// 例如 /api/v1/oss/configs/2 => oss_config
	// 例如 /api/v1/user/current => user
	if len(parts) >= 4 {
		if parts[3] == "oss" {
			if len(parts) >= 5 {
				if parts[4] == "files" {
					return "oss_file"
				}
				if parts[4] == "configs" {
					return "oss_config"
				}
				if parts[4] == "multipart" {
					return "oss_multipart"
				}
			}
			return "oss"
		}
		if parts[3] == "user" {
			return "user"
		}
		if parts[3] == "auth" {
			return "auth"
		}
	}

	return parts[len(parts)-2]
}

// getResourceID 从请求路径中获取资源ID
func getResourceID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return ""
	}

	// 获取资源ID，通常是最后一部分
	// 例如 /api/v1/oss/files/1 => 1
	lastPart := parts[len(parts)-1]

	// 如果最后一部分是操作，则ID是倒数第二部分
	// 例如 /api/v1/oss/configs/1/default => 1
	if lastPart == "default" || lastPart == "download" {
		if len(parts) >= 5 {
			return parts[len(parts)-2]
		}
	}

	// 检查最后一部分是否看起来像ID（数字）
	if lastPart != "files" && lastPart != "configs" && lastPart != "multipart" && lastPart != "current" {
		return lastPart
	}

	return ""
}
