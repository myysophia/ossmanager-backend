package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/auth"
	"github.com/myysophia/ossmanager-backend/internal/config"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"go.uber.org/zap"
	"strings"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 JWT 令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, utils.CodeUnauthorized, "未提供认证令牌")
			c.Abort()
			return
		}

		// 检查 Authorization 头格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			utils.Error(c, utils.CodeUnauthorized, "认证令牌格式错误")
			c.Abort()
			return
		}

		// 获取JWT配置
		jwtConfig := &config.JWTConfig{
			SecretKey: "your-secret-key", // 这应该从配置中读取
			ExpiresIn: 3600,              // 过期时间，单位：秒
			Issuer:    "oss-manager-backend",
		}

		// 解析JWT令牌
		claims, err := auth.ParseToken(parts[1], jwtConfig)
		if err != nil {
			logger.Warn("解析JWT令牌失败", zap.Error(err))
			utils.Error(c, utils.CodeUnauthorized, "无效的认证令牌")
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
