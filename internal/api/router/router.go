package router

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/api/handlers"
	"github.com/myysophia/ossmanager-backend/internal/api/middleware"
)

// SetupRouter 设置路由
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 使用中间件
	r.Use(middleware.Cors())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())

	// API v1 路由组
	v1 := r.Group("/api/v1")
	{
		// 认证相关路由
		authHandler := handlers.NewAuthHandler()
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// 需要认证的路由组
		authenticated := v1.Group("")
		authenticated.Use(middleware.JWT())
		{
			// 用户管理路由
			userHandler := handlers.NewUserHandler()
			users := authenticated.Group("/users")
			{
				users.GET("", userHandler.List)
				users.POST("", userHandler.Create)
				users.GET("/:id", userHandler.Get)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
			}

			// 角色管理路由
			roleHandler := handlers.NewRoleHandler()
			roles := authenticated.Group("/roles")
			{
				roles.GET("", roleHandler.List)
				roles.POST("", roleHandler.Create)
				roles.GET("/:id", roleHandler.Get)
				roles.PUT("/:id", roleHandler.Update)
				roles.DELETE("/:id", roleHandler.Delete)
			}

			// 权限管理路由
			permissionHandler := handlers.NewPermissionHandler()
			permissions := authenticated.Group("/permissions")
			{
				permissions.GET("", permissionHandler.List)
				permissions.POST("", permissionHandler.Create)
				permissions.GET("/:id", permissionHandler.Get)
				permissions.PUT("/:id", permissionHandler.Update)
				permissions.DELETE("/:id", permissionHandler.Delete)
			}

			// 当前用户信息
			authenticated.GET("/user/current", authHandler.GetCurrentUser)
			authenticated.PUT("/user/password", authHandler.UpdatePassword)

			// OSS文件管理路由
			ossFileHandler := handlers.NewOSSFileHandler()
			ossFiles := authenticated.Group("/oss/files")
			{
				ossFiles.POST("", ossFileHandler.Upload)
				ossFiles.GET("", ossFileHandler.List)
				ossFiles.DELETE("/:id", ossFileHandler.Delete)
				ossFiles.GET("/:id/download", ossFileHandler.GetDownloadURL)
				ossFiles.POST("/:id/md5", ossFileHandler.TriggerMD5Calculation)
				ossFiles.GET("/:id/md5", ossFileHandler.GetMD5)
			}

			// OSS配置管理路由
			ossConfigHandler := handlers.NewOSSConfigHandler()
			ossConfigs := authenticated.Group("/oss/configs")
			{
				ossConfigs.POST("", ossConfigHandler.Create)
				ossConfigs.GET("", ossConfigHandler.List)
				ossConfigs.GET("/:id", ossConfigHandler.Get)
				ossConfigs.PUT("/:id", ossConfigHandler.Update)
				ossConfigs.DELETE("/:id", ossConfigHandler.Delete)
				ossConfigs.PUT("/:id/default", ossConfigHandler.SetDefault)
				ossConfigs.POST("/:id/test", ossConfigHandler.TestConnection)
			}

			// 分片上传路由
			ossMultipartHandler := handlers.NewOSSMultipartHandler()
			ossMultipart := authenticated.Group("/oss/multipart")
			{
				ossMultipart.POST("/init", ossMultipartHandler.InitMultipartUpload)
				ossMultipart.POST("/complete", ossMultipartHandler.CompleteMultipartUpload)
				ossMultipart.DELETE("/abort", ossMultipartHandler.AbortMultipartUpload)
			}

			// 审计日志路由
			auditLogHandler := handlers.NewAuditLogHandler()
			auditLogs := authenticated.Group("/audit/logs")
			{
				auditLogs.GET("", auditLogHandler.List)
			}
		}
	}

	return r
}
