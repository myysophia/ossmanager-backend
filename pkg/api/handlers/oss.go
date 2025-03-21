package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/pkg/db"
	"github.com/myysophia/ossmanager-backend/pkg/db/models"
	"github.com/myysophia/ossmanager-backend/pkg/oss"
	"github.com/myysophia/ossmanager-backend/pkg/oss/factory"
	"io"
	"mime/multipart"
	"path"
	"strconv"
	"time"
)

// OSSHandler OSS处理器
type OSSHandler struct {
	*BaseHandler
	storageFactory *factory.DefaultStorageFactory
}

// NewOSSHandler 创建OSS处理器
func NewOSSHandler(storageFactory *factory.DefaultStorageFactory) *OSSHandler {
	return &OSSHandler{
		BaseHandler:    NewBaseHandler(),
		storageFactory: storageFactory,
	}
}

// UploadFile 上传文件
func (h *OSSHandler) UploadFile(c *gin.Context) {
	// 获取文件
	file, err := c.FormFile("file")
	if err != nil {
		h.BadRequest(c, "获取文件失败")
		return
	}

	// 获取存储类型
	storageType := c.DefaultPostForm("storage_type", "")
	if storageType == "" {
		// 使用默认存储服务
		service, err := h.storageFactory.GetDefaultStorageService()
		if err != nil {
			h.InternalError(c, "获取默认存储服务失败")
			return
		}
		storageType = service.GetType()
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(storageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		h.InternalError(c, "打开文件失败")
		return
	}
	defer src.Close()

	// 生成文件名
	filename := generateFilename(file.Filename)

	// 上传文件
	url, err := service.Upload(src, filename)
	if err != nil {
		h.InternalError(c, "上传文件失败")
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		Filename:     filename,
		OriginalName: file.Filename,
		Size:         file.Size,
		MimeType:     file.Header.Get("Content-Type"),
		URL:          url,
		StorageType:  storageType,
	}

	if err := db.GetDB().Create(&ossFile).Error; err != nil {
		h.InternalError(c, "保存文件记录失败")
		return
	}

	h.Success(c, ossFile)
}

// InitMultipartUpload 初始化分片上传
func (h *OSSHandler) InitMultipartUpload(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		h.BadRequest(c, "文件名不能为空")
		return
	}

	storageType := c.DefaultQuery("storage_type", "")
	if storageType == "" {
		// 使用默认存储服务
		service, err := h.storageFactory.GetDefaultStorageService()
		if err != nil {
			h.InternalError(c, "获取默认存储服务失败")
			return
		}
		storageType = service.GetType()
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(storageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 初始化分片上传
	uploadID, urls, err := service.InitMultipartUpload(filename)
	if err != nil {
		h.InternalError(c, "初始化分片上传失败")
		return
	}

	h.Success(c, gin.H{
		"upload_id": uploadID,
		"urls":      urls,
	})
}

// CompleteMultipartUpload 完成分片上传
func (h *OSSHandler) CompleteMultipartUpload(c *gin.Context) {
	var req struct {
		UploadID string     `json:"upload_id" binding:"required"`
		Filename string     `json:"filename" binding:"required"`
		Parts    []oss.Part `json:"parts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "请求参数错误")
		return
	}

	storageType := c.DefaultQuery("storage_type", "")
	if storageType == "" {
		// 使用默认存储服务
		service, err := h.storageFactory.GetDefaultStorageService()
		if err != nil {
			h.InternalError(c, "获取默认存储服务失败")
			return
		}
		storageType = service.GetType()
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(storageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 完成分片上传
	url, err := service.CompleteMultipartUpload(req.UploadID, req.Parts, req.Filename)
	if err != nil {
		h.InternalError(c, "完成分片上传失败")
		return
	}

	// 获取文件大小
	size, err := service.GetObjectInfo(req.Filename)
	if err != nil {
		h.InternalError(c, "获取文件信息失败")
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		Filename:     req.Filename,
		OriginalName: req.Filename,
		Size:         size,
		MimeType:     "application/octet-stream", // 这里应该根据实际情况设置
		URL:          url,
		StorageType:  storageType,
	}

	if err := db.GetDB().Create(&ossFile).Error; err != nil {
		h.InternalError(c, "保存文件记录失败")
		return
	}

	h.Success(c, ossFile)
}

// AbortMultipartUpload 取消分片上传
func (h *OSSHandler) AbortMultipartUpload(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		h.BadRequest(c, "上传ID不能为空")
		return
	}

	filename := c.Query("filename")
	if filename == "" {
		h.BadRequest(c, "文件名不能为空")
		return
	}

	storageType := c.DefaultQuery("storage_type", "")
	if storageType == "" {
		// 使用默认存储服务
		service, err := h.storageFactory.GetDefaultStorageService()
		if err != nil {
			h.InternalError(c, "获取默认存储服务失败")
			return
		}
		storageType = service.GetType()
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(storageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 取消分片上传
	if err := service.AbortMultipartUpload(uploadID, filename); err != nil {
		h.InternalError(c, "取消分片上传失败")
		return
	}

	h.Success(c, nil)
}

// GetFileList 获取文件列表
func (h *OSSHandler) GetFileList(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 查询文件列表
	var files []models.OSSFile
	var total int64

	query := db.GetDB().Model(&models.OSSFile{})

	// 按存储类型筛选
	if storageType := c.Query("storage_type"); storageType != "" {
		query = query.Where("storage_type = ?", storageType)
	}

	// 按文件名搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("filename LIKE ? OR original_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		h.InternalError(c, "获取文件总数失败")
		return
	}

	// 获取分页数据
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error; err != nil {
		h.InternalError(c, "获取文件列表失败")
		return
	}

	h.Success(c, gin.H{
		"total": total,
		"items": files,
	})
}

// DeleteFile 删除文件
func (h *OSSHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		h.BadRequest(c, "文件ID不能为空")
		return
	}

	// 查询文件信息
	var file models.OSSFile
	if err := db.GetDB().First(&file, fileID).Error; err != nil {
		h.NotFound(c, "文件不存在")
		return
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(file.StorageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 删除文件
	if err := service.DeleteObject(file.Filename); err != nil {
		h.InternalError(c, "删除文件失败")
		return
	}

	// 删除文件记录
	if err := db.GetDB().Delete(&file).Error; err != nil {
		h.InternalError(c, "删除文件记录失败")
		return
	}

	h.Success(c, nil)
}

// GenerateDownloadURL 生成下载URL
func (h *OSSHandler) GenerateDownloadURL(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		h.BadRequest(c, "文件ID不能为空")
		return
	}

	// 查询文件信息
	var file models.OSSFile
	if err := db.GetDB().First(&file, fileID).Error; err != nil {
		h.NotFound(c, "文件不存在")
		return
	}

	// 获取存储服务
	service, err := h.storageFactory.GetStorageService(file.StorageType)
	if err != nil {
		h.InternalError(c, "获取存储服务失败")
		return
	}

	// 生成下载URL
	expiration := 24 * time.Hour // 默认24小时
	if exp := c.Query("expiration"); exp != "" {
		if expInt, err := strconv.Atoi(exp); err == nil {
			expiration = time.Duration(expInt) * time.Hour
		}
	}

	url, expires, err := service.GenerateDownloadURL(file.Filename, expiration)
	if err != nil {
		h.InternalError(c, "生成下载URL失败")
		return
	}

	h.Success(c, gin.H{
		"url":     url,
		"expires": expires,
	})
}

// generateFilename 生成文件名
func generateFilename(originalName string) string {
	ext := path.Ext(originalName)
	return time.Now().Format("20060102150405") + ext
}
