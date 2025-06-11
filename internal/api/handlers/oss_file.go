package handlers

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/myysophia/ossmanager-backend/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/oss"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"gorm.io/gorm"
)

type OSSFileHandler struct {
	*BaseHandler
	storageFactory oss.StorageFactory
	DB             *gorm.DB
}

func NewOSSFileHandler(storageFactory oss.StorageFactory, db *gorm.DB) *OSSFileHandler {
	return &OSSFileHandler{
		BaseHandler:    NewBaseHandler(),
		storageFactory: storageFactory,
		DB:             db,
	}
}

// Upload 上传文件
func (h *OSSFileHandler) Upload(c *gin.Context) {
	// 获取用户ID
	userID := c.GetUint("userID")

	// 获取用户指定的 bucket 信息
	regionCode := c.GetHeader("region_code")
	bucketName := c.GetHeader("bucket_name")

	if regionCode == "" || bucketName == "" {
		h.Error(c, utils.CodeInvalidParams, "请指定 region_code 和 bucket_name")
		return
	}

	// 获取存储配置
	var config models.OSSConfig
	if err := h.DB.Where("is_default = ?", true).First(&config).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取默认存储配置失败")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, userID, regionCode, bucketName) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		h.Error(c, utils.CodeInvalidParams, "获取文件失败")
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
	username, _ := c.Get("username")
	objectKey := utils.GenerateObjectKey(username.(string), ext)

	// 上传文件
	src, err := file.Open()
	if err != nil {
		h.Error(c, utils.CodeServerError, "打开文件失败")
		return
	}
	defer src.Close()

	// 使用用户指定的 bucket 上传
	uploadURL, err := storage.UploadToBucket(src, objectKey, regionCode, bucketName)
	if err != nil {
		h.Error(c, utils.CodeServerError, "上传文件失败")
		return
	}

	// 从配置中获取过期时间，如果未配置则默认为24小时
	expireTime := config.URLExpireTime
	if expireTime <= 0 {
		expireTime = 24 * 3600 // 默认24小时
	}
	expiresAt := time.Now().Add(time.Duration(expireTime) * time.Second)

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:         config.ID,
		Filename:         objectKey,
		OriginalFilename: file.Filename,
		FileSize:         file.Size,
		StorageType:      config.StorageType,
		Bucket:           bucketName,
		ObjectKey:        objectKey,
		DownloadURL:      uploadURL,
		UploaderID:       utils.GetUserID(c),
		UploadIP:         c.ClientIP(),
		ExpiresAt:        expiresAt,
		Status:           "ACTIVE",
	}

	if err := h.DB.Create(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	// 上传成功后，触发MD5计算
	//go h.triggerMD5Calculation(ossFile.ID)

	h.Success(c, ossFile)
}

// InitMultipartUpload 初始化分片上传
func (h *OSSFileHandler) InitMultipartUpload(c *gin.Context) {
	var req struct {
		RegionCode string `json:"region_code" binding:"required"`
		BucketName string `json:"bucket_name" binding:"required"`
		FileName   string `json:"file_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, utils.CodeInvalidParams, "参数错误")
		return
	}

	// 获取存储配置
	var config models.OSSConfig
	if err := h.DB.Where("is_default = ?", true).First(&config).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取默认存储配置失败")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), req.RegionCode, req.BucketName) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	ext := filepath.Ext(req.FileName)
	username, _ := c.Get("username")
	objectKey := utils.GenerateObjectKey(username.(string), ext)

	uploadID, urls, err := storage.InitMultipartUploadToBucket(objectKey, req.RegionCode, req.BucketName)
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
		RegionCode string   `json:"region_code" binding:"required"`
		BucketName string   `json:"bucket_name" binding:"required"`
		ObjectKey  string   `json:"object_key" binding:"required"`
		UploadID   string   `json:"upload_id" binding:"required"`
		Parts      []string `json:"parts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.Error(c, utils.CodeInvalidParams, "参数错误")
		return
	}

	// 获取存储配置
	var config models.OSSConfig
	if err := h.DB.Where("is_default = ?", true).First(&config).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取默认存储配置失败")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), req.RegionCode, req.BucketName) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
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

	// 完成分片上传
	url, err := storage.CompleteMultipartUploadToBucket(req.ObjectKey, req.UploadID, ossParts, req.RegionCode, req.BucketName)
	if err != nil {
		h.Error(c, utils.CodeServerError, "完成分片上传失败")
		return
	}

	// 保存文件记录
	ossFile := models.OSSFile{
		ConfigID:         config.ID,
		Filename:         req.ObjectKey,
		OriginalFilename: req.ObjectKey, // 这里可能需要前端传入原始文件名
		StorageType:      config.StorageType,
		Bucket:           req.BucketName,
		ObjectKey:        req.ObjectKey,
		DownloadURL:      url,
		UploaderID:       utils.GetUserID(c),
		UploadIP:         c.ClientIP(),
		Status:           "ACTIVE",
	}

	if err := h.DB.Create(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	// 上传成功后，触发MD5计算
	//go h.triggerMD5Calculation(ossFile.ID)

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
	if err := h.DB.First(&config, req.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), config.Region, config.Bucket) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
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

// List 获取文件列表，相同文件名只获取最新一个
func (h *OSSFileHandler) List(c *gin.Context) {
	// 获取用户ID
	userID := c.GetUint("userID")

	// 获取用户可访问的桶列表
	buckets, err := auth.GetUserAccessibleBuckets(h.DB, userID, "")
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取可访问桶列表失败")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	configID := c.Query("config_id")
	// 首先，获取去重后的所有文件名
	var uniqueFileNames []string
	query := h.DB.Model(&models.OSSFile{}).Select("DISTINCT original_filename").Where("bucket IN ?", buckets)
	if configID != "" {
		query = query.Where("config_id = ?", configID)
	}

	if err := query.Pluck("original_filename", &uniqueFileNames).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取唯一文件名失败")
		return
	}

	total := int64(len(uniqueFileNames))

	// 对于分页的处理
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	if startIdx >= len(uniqueFileNames) {
		h.Success(c, gin.H{
			"total": total,
			"items": []models.OSSFile{},
		})
		return
	}
	if endIdx > len(uniqueFileNames) {
		endIdx = len(uniqueFileNames)
	}

	// 获取当前页的文件名
	pageFileNames := uniqueFileNames[startIdx:endIdx]

	// 对于每个文件名，获取最新的记录
	var files []models.OSSFile
	for _, fileName := range pageFileNames {
		var latest models.OSSFile
		subQuery := h.DB.Model(&models.OSSFile{}).Where("original_filename = ? AND bucket IN ?", fileName, buckets)
		if configID != "" {
			subQuery = subQuery.Where("config_id = ?", configID)
		}

		if err := subQuery.Order("created_at DESC").First(&latest).Error; err != nil {
			// 如果查询出错，跳过这个文件名
			continue
		}

		files = append(files, latest)
	}

	h.Success(c, gin.H{
		"total": total,
		"items": files,
	})
}

// Delete 删除文件
func (h *OSSFileHandler) Delete(c *gin.Context) {
	// 获取用户ID
	userID := c.GetUint("userID")

	// 获取文件信息
	var file models.OSSFile
	if err := h.DB.First(&file, c.Param("id")).Error; err != nil {
		h.Error(c, utils.CodeFileNotFound, "文件不存在")
		return
	}

	// 获取配置信息以获取Region
	var config models.OSSConfig
	if err := h.DB.First(&config, file.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, userID, file.DownloadURL, file.Bucket) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
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

	if err := h.DB.Delete(&file).Error; err != nil {
		h.Error(c, utils.CodeServerError, "删除文件记录失败")
		return
	}

	h.Success(c, nil)
}

// GetDownloadURL 获取文件下载链接
func (h *OSSFileHandler) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("id")

	var file models.OSSFile
	if err := h.DB.First(&file, fileID).Error; err != nil {
		h.Error(c, utils.CodeFileNotFound, "文件不存在")
		return
	}

	// 获取配置信息以获取Region
	var config models.OSSConfig
	if err := h.DB.First(&config, file.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), file.DownloadURL, file.Bucket) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
		return
	}

	storage, err := h.storageFactory.GetStorageService(config.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	// 动态生成下载链接，传递 bucket 信息
	var downloadURL string
	var expires time.Time
	if aliyunStorage, ok := storage.(*oss.AliyunOSSService); ok {
		downloadURL, expires, err = aliyunStorage.GenerateDownloadURLWithBucket(file.ObjectKey, file.DownloadURL, 1*time.Hour)
		if err != nil {
			h.Error(c, utils.CodeServerError, "生成下载链接失败")
			return
		}
	} else {
		downloadURL, expires, err = storage.GenerateDownloadURL(file.ObjectKey, 24*time.Hour)
		if err != nil {
			h.Error(c, utils.CodeServerError, "生成下载链接失败")
			return
		}
	}

	h.Success(c, gin.H{
		"download_url": downloadURL,
		"expires":      expires,
	})
}

// GetByOriginalFilename 根据原始文件名获取文件详情
//func (h *OSSFileHandler) GetByOriginalFilename(c *gin.Context) {
//	filename := c.Query("filename")
//	if filename == "" {
//		h.Error(c, utils.CodeInvalidParams, "文件名不能为空")
//		return
//	}
//
//	var ossFile models.OSSFile
//	if err := h.DB.Where("original_filename = ? AND status = ?", filename, "ACTIVE").First(&ossFile).Error; err != nil {
//		h.Error(c, utils.CodeNotFound, "文件不存在")
//		return
//	}
//
//	// 获取配置信息以获取Region
//	var config models.OSSConfig
//	if err := h.DB.First(&config, ossFile.ConfigID).Error; err != nil {
//		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
//		return
//	}
//
//	// 检查用户是否有权限访问该桶
//	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), config.Region, ossFile.Bucket) {
//		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
//		return
//	}
//
//	storage, err := h.storageFactory.GetStorageService(ossFile.StorageType)
//	if err != nil {
//		h.Error(c, utils.CodeServerError, "获取存储服务失败")
//		return
//	}
//
//	downloadURL, expires, err := storage.GenerateDownloadURL(ossFile.ObjectKey, 24*time.Hour)
//	if err != nil {
//		h.Error(c, utils.CodeServerError, "生成下载链接失败")
//		return
//	}
//
//	// 更新下载URL和过期时间
//	ossFile.DownloadURL = downloadURL
//	ossFile.ExpiresAt = expires
//	if err := h.DB.Save(&ossFile).Error; err != nil {
//		logger.Error("更新文件下载URL失败", zap.Error(err))
//	}
//
//	h.Success(c, ossFile)
//}
