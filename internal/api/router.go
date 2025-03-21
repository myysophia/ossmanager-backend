package api

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/api/handlers"
	"github.com/myysophia/ossmanager-backend/internal/api/middleware"
	"github.com/myysophia/ossmanager-backend/internal/function"
	"github.com/myysophia/ossmanager-backend/internal/oss"
)

// SetupRouter 设置路由
func SetupRouter(storageFactory *oss.DefaultStorageFactory, md5Calculator *function.MD5Calculator) *gin.Engine {
	// 创建Gin实例
	router := gin.New()

	// 全局中间件
	router.Use(
		gin.Recovery(),                  // 内置恢复中间件
		middleware.RecoveryMiddleware(), // 自定义恢复中间件
		middleware.LoggerMiddleware(),   // 日志中间件
		middleware.CorsMiddleware(),     // 跨域中间件
	)

	// 创建处理器
	authHandler := handlers.NewAuthHandler()
	ossFileHandler := handlers.NewOSSFileHandler(storageFactory)
	ossConfigHandler := handlers.NewOSSConfigHandler(storageFactory)
	md5Handler := handlers.NewMD5Handler(md5Calculator)

	// 公开路由
	public := router.Group("/api/v1")
	{
		// 认证相关
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/register", authHandler.Register)
	}

	// 需要认证的路由
	authorized := router.Group("/api/v1")
	authorized.Use(
		middleware.AuthMiddleware(),     // 认证中间件
		middleware.AuditLogMiddleware(), // 审计日志中间件
	)
	{
		// 用户相关
		authorized.GET("/user/current", authHandler.GetCurrentUser)

		// OSS文件管理
		authorized.POST("/oss/files", ossFileHandler.Upload)
		authorized.GET("/oss/files", ossFileHandler.List)
		authorized.DELETE("/oss/files/:id", ossFileHandler.Delete)
		authorized.GET("/oss/files/:id/download", ossFileHandler.GetDownloadURL)

		// 分片上传
		authorized.POST("/oss/multipart/init", ossFileHandler.InitMultipartUpload)
		authorized.POST("/oss/multipart/complete", ossFileHandler.CompleteMultipartUpload)
		authorized.DELETE("/oss/multipart/abort", ossFileHandler.AbortMultipartUpload)

		// MD5计算相关
		authorized.POST("/oss/files/:id/md5", md5Handler.TriggerCalculation)
		authorized.GET("/oss/files/:id/md5", md5Handler.GetMD5)

		// OSS配置管理（仅管理员可访问）
		configs := authorized.Group("/oss/configs")
		configs.Use(middleware.AdminMiddleware()) // 管理员权限中间件
		{
			configs.POST("", ossConfigHandler.CreateConfig)
			configs.PUT("/:id", ossConfigHandler.UpdateConfig)
			configs.DELETE("/:id", ossConfigHandler.DeleteConfig)
			configs.GET("", ossConfigHandler.GetConfigList)
			configs.GET("/:id", ossConfigHandler.GetConfig)
			configs.PUT("/:id/default", ossConfigHandler.SetDefaultConfig)
		}
	}

	return router
}
