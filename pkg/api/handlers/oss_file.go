package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/pkg/db"
	"github.com/myysophia/ossmanager-backend/pkg/db/models"
	"github.com/myysophia/ossmanager-backend/pkg/oss"
	"github.com/myysophia/ossmanager-backend/pkg/utils"
	"github.com/myysophia/ossmanager-backend/pkg/utils/response"
	"path/filepath"
	"strconv"
)

type OSSFileHandler struct {
	db             *db.DB
	storageFactory *oss.StorageFactory
}

func NewOSSFileHandler(db *db.DB, storageFactory *oss.StorageFactory) *OSSFileHandler {
	return &OSSFileHandler{
		db:             db,
		storageFactory: storageFactory,
	}
}

// Upload 上传文件
func (h *OSSFileHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, response.CodeInvalidParams, "获取文件失败", err)
		return
	}

	// 获取存储配置
	configID := c.PostForm("config_id")
	if configID == "" {
		response.Error(c, response.CodeInvalidParams, "存储配置ID不能为空", nil)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	// 获取存储服务
	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	// 生成文件路径
	ext := filepath.Ext(file.Filename)
	objectKey := utils.GenerateObjectKey(ext)

	// 上传文件
	uploadURL, err := storage.Upload(c.Request.Context(), objectKey, file)
	if err != nil {
		response.Error(c, response.CodeServerError, "上传文件失败", err)
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:   config.ID,
		ConfigName: config.Name,
		ObjectKey:  objectKey,
		FileName:   file.Filename,
		FileSize:   file.Size,
		FileType:   file.Header.Get("Content-Type"),
		UploadURL:  uploadURL,
		CreatedBy:  utils.GetUserID(c),
	}

	if err := h.db.Create(&ossFile).Error; err != nil {
		response.Error(c, response.CodeServerError, "保存文件记录失败", err)
		return
	}

	response.Success(c, ossFile)
}

// InitMultipartUpload 初始化分片上传
func (h *OSSFileHandler) InitMultipartUpload(c *gin.Context) {
	var req struct {
		ConfigID string `json:"config_id" binding:"required"`
		FileName string `json:"file_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, req.ConfigID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	ext := filepath.Ext(req.FileName)
	objectKey := utils.GenerateObjectKey(ext)

	uploadID, err := storage.InitMultipartUpload(c.Request.Context(), objectKey)
	if err != nil {
		response.Error(c, response.CodeServerError, "初始化分片上传失败", err)
		return
	}

	response.Success(c, gin.H{
		"upload_id":  uploadID,
		"object_key": objectKey,
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
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, req.ConfigID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	downloadURL, err := storage.CompleteMultipartUpload(c.Request.Context(), req.ObjectKey, req.UploadID, req.Parts)
	if err != nil {
		response.Error(c, response.CodeServerError, "完成分片上传失败", err)
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:   config.ID,
		ConfigName: config.Name,
		ObjectKey:  req.ObjectKey,
		FileName:   filepath.Base(req.ObjectKey),
		UploadURL:  downloadURL,
		CreatedBy:  utils.GetUserID(c),
	}

	if err := h.db.Create(&ossFile).Error; err != nil {
		response.Error(c, response.CodeServerError, "保存文件记录失败", err)
		return
	}

	response.Success(c, ossFile)
}

// AbortMultipartUpload 取消分片上传
func (h *OSSFileHandler) AbortMultipartUpload(c *gin.Context) {
	var req struct {
		ConfigID  string `json:"config_id" binding:"required"`
		ObjectKey string `json:"object_key" binding:"required"`
		UploadID  string `json:"upload_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, req.ConfigID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	if err := storage.AbortMultipartUpload(c.Request.Context(), req.ObjectKey, req.UploadID); err != nil {
		response.Error(c, response.CodeServerError, "取消分片上传失败", err)
		return
	}

	response.Success(c, nil)
}

// List 获取文件列表
func (h *OSSFileHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	configID := c.Query("config_id")

	query := h.db.Model(&models.OSSFile{})
	if configID != "" {
		query = query.Where("config_id = ?", configID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Error(c, response.CodeServerError, "获取文件总数失败", err)
		return
	}

	var files []models.OSSFile
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error; err != nil {
		response.Error(c, response.CodeServerError, "获取文件列表失败", err)
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"items": files,
	})
}

// Delete 删除文件
func (h *OSSFileHandler) Delete(c *gin.Context) {
	fileID := c.Param("id")

	var file models.OSSFile
	if err := h.db.First(&file, fileID).Error; err != nil {
		response.Error(c, response.CodeFileNotFound, "文件不存在", nil)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, file.ConfigID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	if err := storage.DeleteObject(c.Request.Context(), file.ObjectKey); err != nil {
		response.Error(c, response.CodeServerError, "删除文件失败", err)
		return
	}

	if err := h.db.Delete(&file).Error; err != nil {
		response.Error(c, response.CodeServerError, "删除文件记录失败", err)
		return
	}

	response.Success(c, nil)
}

// GetDownloadURL 获取文件下载链接
func (h *OSSFileHandler) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("id")

	var file models.OSSFile
	if err := h.db.First(&file, fileID).Error; err != nil {
		response.Error(c, response.CodeFileNotFound, "文件不存在", nil)
		return
	}

	var config models.OSSConfig
	if err := h.db.First(&config, file.ConfigID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	storage, err := h.storageFactory.GetStorage(config.Type)
	if err != nil {
		response.Error(c, response.CodeServerError, "获取存储服务失败", err)
		return
	}

	downloadURL, err := storage.GenerateDownloadURL(c.Request.Context(), file.ObjectKey)
	if err != nil {
		response.Error(c, response.CodeServerError, "生成下载链接失败", err)
		return
	}

	response.Success(c, gin.H{
		"download_url": downloadURL,
	})
}
