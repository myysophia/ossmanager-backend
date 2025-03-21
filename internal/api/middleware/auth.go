package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/auth"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"github.com/myysophia/ossmanager-backend/internal/utils/response"
	"go.uber.org/zap"
	"strings"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 JWT 令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, response.CodeUnauthorized, "未提供认证令牌")
			c.Abort()
			return
		}

		// 检查 Authorization 头格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Error(c, response.CodeUnauthorized, "认证令牌格式错误")
			c.Abort()
			return
		}

		// 解析令牌
		token := parts[1]
		claims, err := auth.ParseToken(token)
		if err != nil {
			logger.Error("解析令牌失败", zap.Error(err))
			response.Error(c, response.CodeUnauthorized, "无效的认证令牌")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}
