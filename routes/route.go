package routes

import (
	"github.com/gin-gonic/gin"
	"oss-backend/controllers"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/login", controllers.Login)
	r.POST("/upload", controllers.UploadFile)

	return r
}
