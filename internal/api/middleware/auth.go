package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/auth"
	"github.com/myysophia/ossmanager-backend/internal/config"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"go.uber.org/zap"
)

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 JWT 令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ResponseError(c, utils.CodeUnauthorized, errors.New("未提供认证令牌"))
			c.Abort()
			return
		}

		// 检查 Authorization 头格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			utils.ResponseError(c, utils.CodeUnauthorized, errors.New("认证令牌格式错误"))
			c.Abort()
			return
		}

		// 获取JWT配置
		jwtConfig := config.GetConfig().JWT
		logger.Debug("JWT配置", zap.String("issuer", jwtConfig.Issuer), zap.Int("expiresIn", jwtConfig.ExpiresIn))

		// 解析JWT令牌
		claims, err := auth.ParseToken(parts[1], &jwtConfig)
		if err != nil {
			logger.Warn("解析JWT令牌失败", zap.Error(err))
			utils.ResponseError(c, utils.CodeUnauthorized, errors.New("无效的认证令牌"))
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
