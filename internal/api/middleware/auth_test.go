package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/auth/jwt"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	// 设置测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试路由
	r := gin.New()
	r.Use(AuthMiddleware())

	// 创建测试处理器
	r.GET("/test", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		assert.True(t, exists)
		assert.Equal(t, uint(1), userID)

		username, exists := c.Get("username")
		assert.True(t, exists)
		assert.Equal(t, "test", username)

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试有效token
	t.Run("Valid Token", func(t *testing.T) {
		// 生成测试token
		token, err := jwt.GenerateToken(1, "test")
		assert.NoError(t, err)

		// 创建测试请求
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 测试无效token
	t.Run("Invalid Token", func(t *testing.T) {
		// 创建测试请求
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		r.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// 测试缺少token
	t.Run("Missing Token", func(t *testing.T) {
		// 创建测试请求
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
