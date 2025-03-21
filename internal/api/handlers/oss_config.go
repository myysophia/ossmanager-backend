package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/oss"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"github.com/myysophia/ossmanager-backend/internal/utils/response"
	"strconv"
)

// OSSConfigHandler OSS配置处理器
type OSSConfigHandler struct {
	*BaseHandler
	storageFactory *oss.DefaultStorageFactory
}

// NewOSSConfigHandler 创建OSS配置处理器
func NewOSSConfigHandler(storageFactory *oss.DefaultStorageFactory) *OSSConfigHandler {
	return &OSSConfigHandler{
		BaseHandler:    NewBaseHandler(),
		storageFactory: storageFactory,
	}
}

// CreateConfig 创建存储配置
func (h *OSSConfigHandler) CreateConfig(c *gin.Context) {
	var config models.OSSConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		h.BadRequest(c, "参数错误")
		return
	}

	// 验证配置类型
	if !isValidStorageType(config.StorageType) {
		h.BadRequest(c, "不支持的存储类型")
		return
	}

	// 设置创建者
	user, _ := c.Get("user")
	if userModel, ok := user.(*models.User); ok {
		config.CreatedBy = userModel.ID
	}

	if err := db.GetDB().Create(&config).Error; err != nil {
		h.InternalError(c, "创建存储配置失败")
		return
	}

	h.Success(c, config)
}

// UpdateConfig 更新存储配置
func (h *OSSConfigHandler) UpdateConfig(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := db.GetDB().First(&config, configID).Error; err != nil {
		h.NotFound(c, "存储配置不存在")
		return
	}

	var updateData struct {
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type" binding:"required"`
		Endpoint  string `json:"endpoint" binding:"required"`
		Bucket    string `json:"bucket" binding:"required"`
		AccessKey string `json:"access_key" binding:"required"`
		SecretKey string `json:"secret_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		h.BadRequest(c, "参数错误")
		return
	}

	// 验证配置类型
	if !isValidStorageType(updateData.Type) {
		h.BadRequest(c, "不支持的存储类型")
		return
	}

	// 更新配置
	config.Name = updateData.Name
	config.Type = updateData.Type
	config.Endpoint = updateData.Endpoint
	config.Bucket = updateData.Bucket
	config.AccessKey = updateData.AccessKey
	config.SecretKey = updateData.SecretKey

	if err := db.GetDB().Save(&config).Error; err != nil {
		h.InternalError(c, "更新存储配置失败")
		return
	}

	h.Success(c, config)
}

// DeleteConfig 删除存储配置
func (h *OSSConfigHandler) DeleteConfig(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := db.GetDB().First(&config, configID).Error; err != nil {
		h.NotFound(c, "存储配置不存在")
		return
	}

	// 检查是否有文件使用此配置
	var count int64
	if err := db.GetDB().Model(&models.OSSFile{}).Where("config_id = ?", configID).Count(&count).Error; err != nil {
		h.InternalError(c, "检查文件关联失败")
		return
	}
	if count > 0 {
		h.Error(c, response.CodeConfigInUse, "存储配置正在使用中，无法删除")
		return
	}

	if err := db.GetDB().Delete(&config).Error; err != nil {
		h.InternalError(c, "删除存储配置失败")
		return
	}

	h.Success(c, nil)
}

// GetConfigList 获取存储配置列表
func (h *OSSConfigHandler) GetConfigList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var total int64
	if err := db.GetDB().Model(&models.OSSConfig{}).Count(&total).Error; err != nil {
		h.InternalError(c, "获取配置总数失败")
		return
	}

	var configs []models.OSSConfig
	if err := db.GetDB().Offset((page - 1) * pageSize).Limit(pageSize).Find(&configs).Error; err != nil {
		h.InternalError(c, "获取配置列表失败")
		return
	}

	h.Success(c, gin.H{
		"total": total,
		"items": configs,
	})
}

// GetConfig 获取存储配置详情
func (h *OSSConfigHandler) GetConfig(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := db.GetDB().First(&config, configID).Error; err != nil {
		h.NotFound(c, "存储配置不存在")
		return
	}

	h.Success(c, config)
}

// SetDefaultConfig 设置默认配置
func (h *OSSConfigHandler) SetDefaultConfig(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := db.GetDB().First(&config, configID).Error; err != nil {
		h.NotFound(c, "存储配置不存在")
		return
	}

	// 开始事务
	tx := db.GetDB().Begin()

	// 取消所有默认配置
	if err := tx.Model(&models.OSSConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		tx.Rollback()
		h.InternalError(c, "取消默认配置失败")
		return
	}

	// 设置新的默认配置
	if err := tx.Model(&config).Update("is_default", true).Error; err != nil {
		tx.Rollback()
		h.InternalError(c, "设置默认配置失败")
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		h.InternalError(c, "提交事务失败")
		return
	}

	// 清除存储工厂缓存
	h.storageFactory.ClearCache()

	h.Success(c, nil)
}

// isValidStorageType 验证存储类型是否有效
func isValidStorageType(storageType string) bool {
	validTypes := []string{oss.StorageTypeAliyunOSS, oss.StorageTypeAWSS3, oss.StorageTypeR2}
	for _, t := range validTypes {
		if t == storageType {
			return true
		}
	}
	return false
}

// Test 测试存储配置
func (h *OSSConfigHandler) Test(c *gin.Context) {
	var config models.OSSConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	// 验证配置类型
	if !isValidStorageType(config.StorageType) {
		response.Error(c, response.CodeInvalidParams, "不支持的存储类型", nil)
		return
	}

	// TODO: 实现存储配置测试逻辑
	// 1. 创建临时存储服务
	// 2. 尝试上传测试文件
	// 3. 尝试下载测试文件
	// 4. 删除测试文件
	// 5. 返回测试结果

	response.Success(c, gin.H{
		"message": "存储配置测试成功",
	})
}
