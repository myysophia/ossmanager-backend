package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/oss"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"path/filepath"
	"strconv"
	"time"
)

type OSSFileHandler struct {
	*BaseHandler
	storageFactory oss.StorageFactory
}

func NewOSSFileHandler(storageFactory oss.StorageFactory) *OSSFileHandler {
	return &OSSFileHandler{
		BaseHandler:    NewBaseHandler(),
		storageFactory: storageFactory,
	}
}

// Upload 上传文件
func (h *OSSFileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		h.Error(c, utils.CodeInvalidParams, "获取文件失败")
		return
	}

	// 获取存储配置
	configID := c.PostForm("config_id")
	if configID == "" {
		h.Error(c, utils.CodeInvalidParams, "存储配置ID不能为空")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, configID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	// 获取存储服务
	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	// 生成文件路径
	ext := filepath.Ext(file.Filename)
	objectKey := utils.GenerateObjectKey(ext)

	// 上传文件
	src, err := file.Open()
	if err != nil {
		h.Error(c, utils.CodeServerError, "打开文件失败")
		return
	}
	defer src.Close()

	uploadURL, err := storage.Upload(src, objectKey)
	if err != nil {
		h.Error(c, utils.CodeServerError, "上传文件失败")
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:         config.ID,
		Filename:         objectKey,
		OriginalFilename: file.Filename,
		FileSize:         file.Size,
		StorageType:      config.StorageType,
		Bucket:           config.Bucket,
		ObjectKey:        objectKey,
		DownloadURL:      uploadURL,
		UploaderID:       utils.GetUserID(c),
		UploadIP:         c.ClientIP(),
		Status:           "ACTIVE",
	}

	if err := db.GetDB().Create(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	h.Success(c, ossFile)
}

// InitMultipartUpload 初始化分片上传
func (h *OSSFileHandler) InitMultipartUpload(c *gin.Context) {
	var req struct {
		ConfigID string `json:"config_id" binding:"required"`
		FileName string `json:"file_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, utils.CodeInvalidParams, "参数错误")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, req.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	ext := filepath.Ext(req.FileName)
	objectKey := utils.GenerateObjectKey(ext)

	uploadID, urls, err := storage.InitMultipartUpload(objectKey)
	if err != nil {
		h.Error(c, utils.CodeServerError, "初始化分片上传失败")
		return
	}

	h.Success(c, gin.H{
		"upload_id":  uploadID,
		"object_key": objectKey,
		"urls":       urls,
	})
}

// CompleteMultipartUpload 完成分片上传
func (h *OSSFileHandler) CompleteMultipartUpload(c *gin.Context) {
	var req struct {
		ConfigID  string   `json:"config_id" binding:"required"`
		ObjectKey string   `json:"object_key" binding:"required"`
		UploadID  string   `json:"upload_id" binding:"required"`
		Parts     []string `json:"parts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, utils.CodeInvalidParams, "参数错误")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, req.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	// 转换parts为oss.Part类型
	ossParts := make([]oss.Part, len(req.Parts))
	for i, part := range req.Parts {
		ossParts[i] = oss.Part{
			PartNumber: i + 1,
			ETag:       part,
		}
	}

	downloadURL, err := storage.CompleteMultipartUpload(req.UploadID, ossParts, req.ObjectKey)
	if err != nil {
		h.Error(c, utils.CodeServerError, "完成分片上传失败")
		return
	}

	// 获取文件大小
	fileSize, err := storage.GetObjectInfo(req.ObjectKey)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取文件信息失败")
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:         config.ID,
		Filename:         req.ObjectKey,
		OriginalFilename: req.ObjectKey,
		FileSize:         fileSize,
		StorageType:      config.StorageType,
		Bucket:           config.Bucket,
		ObjectKey:        req.ObjectKey,
		DownloadURL:      downloadURL,
		UploaderID:       utils.GetUserID(c),
		UploadIP:         c.ClientIP(),
		Status:           "ACTIVE",
	}

	if err := db.GetDB().Create(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	h.Success(c, ossFile)
}

// AbortMultipartUpload 取消分片上传
func (h *OSSFileHandler) AbortMultipartUpload(c *gin.Context) {
	var req struct {
		ConfigID  string `json:"config_id" binding:"required"`
		ObjectKey string `json:"object_key" binding:"required"`
		UploadID  string `json:"upload_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, utils.CodeInvalidParams, "参数错误")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, req.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	if err := storage.AbortMultipartUpload(req.UploadID, req.ObjectKey); err != nil {
		h.Error(c, utils.CodeServerError, "取消分片上传失败")
		return
	}

	h.Success(c, nil)
}

// List 获取文件列表
func (h *OSSFileHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	configID := c.Query("config_id")

	query := db.GetDB().Model(&models.OSSFile{})
	if configID != "" {
		query = query.Where("config_id = ?", configID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取文件总数失败")
		return
	}

	var files []models.OSSFile
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取文件列表失败")
		return
	}

	h.Success(c, gin.H{
		"total": total,
		"items": files,
	})
}

// Delete 删除文件
func (h *OSSFileHandler) Delete(c *gin.Context) {
	fileID := c.Param("id")

	var file models.OSSFile
	if err := db.GetDB().First(&file, fileID).Error; err != nil {
		h.Error(c, utils.CodeFileNotFound, "文件不存在")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, file.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	if err := storage.DeleteObject(file.ObjectKey); err != nil {
		h.Error(c, utils.CodeServerError, "删除文件失败")
		return
	}

	if err := db.GetDB().Delete(&file).Error; err != nil {
		h.Error(c, utils.CodeServerError, "删除文件记录失败")
		return
	}

	h.Success(c, nil)
}

// GetDownloadURL 获取文件下载链接
func (h *OSSFileHandler) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("id")

	var file models.OSSFile
	if err := db.GetDB().First(&file, fileID).Error; err != nil {
		h.Error(c, utils.CodeFileNotFound, "文件不存在")
		return
	}

	var config models.OSSConfig
	if err := db.GetDB().First(&config, file.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	downloadURL, expires, err := storage.GenerateDownloadURL(file.ObjectKey, 24*time.Hour)
	if err != nil {
		h.Error(c, utils.CodeServerError, "生成下载链接失败")
		return
	}

	h.Success(c, gin.H{
		"download_url": downloadURL,
		"expires":      expires,
	})
}
