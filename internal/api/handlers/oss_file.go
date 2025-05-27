package handlers

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/myysophia/ossmanager-backend/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"github.com/myysophia/ossmanager-backend/internal/oss"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"go.uber.org/zap"
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

	// 获取存储配置
	var config models.OSSConfig
	if err := h.DB.Where("is_default = ?", true).First(&config).Error; err != nil {
		h.Error(c, utils.CodeServerError, "获取默认存储配置失败")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, userID, config.Region, config.Bucket) {
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
		Bucket:           config.Bucket,
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
	go h.triggerMD5Calculation(ossFile.ID)

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

	if err := h.DB.Create(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeServerError, "保存文件记录失败")
		return
	}

	// 上传成功后，触发MD5计算
	go h.triggerMD5Calculation(ossFile.ID)

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
	if !auth.CheckBucketAccess(h.DB, userID, config.Region, file.Bucket) {
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
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), config.Region, file.Bucket) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
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

// triggerMD5Calculation 触发MD5计算
func (h *OSSFileHandler) triggerMD5Calculation(fileID uint) {
	// 获取文件信息
	var file models.OSSFile
	if err := h.DB.First(&file, fileID).Error; err != nil {
		logger.Error("获取文件信息失败", zap.Uint("file_id", fileID), zap.Error(err))
		return
	}

	// 获取配置信息以获取Region
	var config models.OSSConfig
	if err := h.DB.First(&config, file.ConfigID).Error; err != nil {
		logger.Error("获取存储配置失败", zap.Uint("config_id", file.ConfigID), zap.Error(err))
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, file.UploaderID, config.Region, file.Bucket) {
		logger.Error("没有权限访问该存储桶",
			zap.Uint("user_id", file.UploaderID),
			zap.String("region", config.Region),
			zap.String("bucket", file.Bucket))
		return
	}

	// 构建请求URL - 使用HTTP方式触发
	//url := fmt.Sprintf("/api/v1/oss/files/%d/md5", fileID)
	//logger.Info("准备触发MD5计算", zap.String("url", url), zap.Uint("file_id", fileID))
	//
	//// 直接调用模型方法计算MD5
	//if file.MD5 != "" {
	//	logger.Info("文件已有MD5值，无需计算", zap.Uint("file_id", fileID), zap.String("md5", file.MD5))
	//	return
	//}
	//
	//// 更新文件状态为计算中
	//fileUpdate := models.OSSFile{
	//	MD5Status: models.MD5StatusCalculating,
	//}
	//if err := h.DB.Model(&models.OSSFile{}).Where("id = ?", fileID).Updates(fileUpdate).Error; err != nil {
	//	logger.Error("更新文件MD5状态失败", zap.Uint("file_id", fileID), zap.Error(err))
	//	return
	//}
	//
	//// 启动一个新的goroutine来计算MD5
	//go func() {
	//	// 查询文件存储配置
	//	var config models.OSSConfig
	//	if err := h.DB.First(&config, file.ConfigID).Error; err != nil {
	//		logger.Error("获取存储配置失败", zap.Uint("config_id", file.ConfigID), zap.Error(err))
	//		h.updateMD5Status(fileID, models.MD5StatusFailed, "")
	//		return
	//	}
	//
	//	// 使用handler中的storageFactory
	//	storage, err := h.storageFactory.GetStorageService(file.StorageType)
	//	if err != nil {
	//		logger.Error("获取存储服务失败", zap.String("storage_type", file.StorageType), zap.Error(err))
	//		h.updateMD5Status(fileID, models.MD5StatusFailed, "")
	//		return
	//	}
	//
	//	// 下载文件并计算MD5
	//	reader, err := storage.GetObject(file.ObjectKey)
	//	if err != nil {
	//		logger.Error("下载文件失败", zap.String("object_key", file.ObjectKey), zap.Error(err))
	//		h.updateMD5Status(fileID, models.MD5StatusFailed, "")
	//		return
	//	}
	//	defer reader.Close()
	//
	//	// 计算MD5
	//	hash := md5.New()
	//	if _, err := io.Copy(hash, reader); err != nil {
	//		logger.Error("计算MD5失败", zap.String("object_key", file.ObjectKey), zap.Error(err))
	//		h.updateMD5Status(fileID, models.MD5StatusFailed, "")
	//		return
	//	}
	//
	//	// 转换为十六进制字符串
	//	md5Str := hex.EncodeToString(hash.Sum(nil))
	//	logger.Info("文件MD5计算完成", zap.Uint("file_id", fileID), zap.String("md5", md5Str))
	//
	//	// 更新MD5
	//	h.updateMD5Status(fileID, models.MD5StatusCompleted, md5Str)
	//}()
	//
	//logger.Info("已触发文件MD5计算", zap.Uint("file_id", fileID))
}

// updateMD5Status 更新文件MD5状态
func (h *OSSFileHandler) updateMD5Status(fileID uint, status string, md5 string) {
	fileUpdate := models.OSSFile{
		MD5Status: status,
		MD5:       md5,
	}
	if err := h.DB.Model(&models.OSSFile{}).Where("id = ?", fileID).Updates(fileUpdate).Error; err != nil {
		logger.Error("更新文件MD5状态失败", zap.Uint("file_id", fileID), zap.Error(err))
	}
}

// GetByOriginalFilename 根据原始文件名获取文件详情
func (h *OSSFileHandler) GetByOriginalFilename(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		h.Error(c, utils.CodeInvalidParams, "文件名不能为空")
		return
	}

	var ossFile models.OSSFile
	if err := h.DB.Where("original_filename = ? AND status = ?", filename, "ACTIVE").First(&ossFile).Error; err != nil {
		h.Error(c, utils.CodeNotFound, "文件不存在")
		return
	}

	// 获取配置信息以获取Region
	var config models.OSSConfig
	if err := h.DB.First(&config, ossFile.ConfigID).Error; err != nil {
		h.Error(c, utils.CodeConfigNotFound, "存储配置不存在")
		return
	}

	// 检查用户是否有权限访问该桶
	if !auth.CheckBucketAccess(h.DB, c.GetUint("userID"), config.Region, ossFile.Bucket) {
		h.Error(c, utils.CodeForbidden, "没有权限访问该存储桶")
		return
	}

	storage, err := h.storageFactory.GetStorageService(ossFile.StorageType)
	if err != nil {
		h.Error(c, utils.CodeServerError, "获取存储服务失败")
		return
	}

	downloadURL, expires, err := storage.GenerateDownloadURL(ossFile.ObjectKey, 24*time.Hour)
	if err != nil {
		h.Error(c, utils.CodeServerError, "生成下载链接失败")
		return
	}

	// 更新下载URL和过期时间
	ossFile.DownloadURL = downloadURL
	ossFile.ExpiresAt = expires
	if err := h.DB.Save(&ossFile).Error; err != nil {
		logger.Error("更新文件下载URL失败", zap.Error(err))
	}

	h.Success(c, ossFile)
}
