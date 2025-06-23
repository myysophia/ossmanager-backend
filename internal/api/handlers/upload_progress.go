package handlers

import (
        "net/http"

        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/myysophia/ossmanager-backend/internal/upload"
)

// UploadProgressHandler 处理上传进度查询和SSE
type UploadProgressHandler struct {
	*BaseHandler
}

func NewUploadProgressHandler() *UploadProgressHandler {
        return &UploadProgressHandler{BaseHandler: NewBaseHandler()}
}

// Init 创建一个新的上传进度任务并返回任务ID
func (h *UploadProgressHandler) Init(c *gin.Context) {
        var req struct {
                Total int64 `json:"total"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                h.BadRequest(c, "参数错误")
                return
        }

        id := uuid.NewString()
        upload.DefaultManager.Start(id, req.Total)
        h.Success(c, gin.H{"id": id})
}

// GetProgress 返回上传进度
func (h *UploadProgressHandler) GetProgress(c *gin.Context) {
	id := c.Param("id")
	if p, ok := upload.DefaultManager.Get(id); ok {
		h.Success(c, p)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"message": "task not found"})
	}
}

// StreamProgress 使用SSE实时推送进度
func (h *UploadProgressHandler) StreamProgress(c *gin.Context) {
	id := c.Param("id")
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	ch := upload.DefaultManager.Subscribe(id)
	defer upload.DefaultManager.Unsubscribe(id, ch)

	notify := c.Writer.CloseNotify()
	for {
		select {
		case p, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent("progress", p)
			c.Writer.Flush()
		case <-notify:
			return
		}
	}
}
