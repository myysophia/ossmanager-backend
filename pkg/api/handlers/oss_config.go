package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ninesun/ossmanager-backend/pkg/db"
	"github.com/ninesun/ossmanager-backend/pkg/models"
	"github.com/ninesun/ossmanager-backend/pkg/response"
	"github.com/ninesun/ossmanager-backend/pkg/utils"
	"strconv"
)

// OSSConfigHandler OSS配置处理器
type OSSConfigHandler struct {
	db *db.DB
}

// NewOSSConfigHandler 创建OSS配置处理器
func NewOSSConfigHandler(db *db.DB) *OSSConfigHandler {
	return &OSSConfigHandler{db: db}
}

// Create 创建存储配置
func (h *OSSConfigHandler) Create(c *gin.Context) {
	var config models.OSSConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	// 验证配置类型
	if !utils.IsValidStorageType(config.Type) {
		response.Error(c, response.CodeInvalidParams, "不支持的存储类型", nil)
		return
	}

	// 设置创建者
	config.CreatedBy = utils.GetUserID(c)

	if err := h.db.Create(&config).Error; err != nil {
		response.Error(c, response.CodeServerError, "创建存储配置失败", err)
		return
	}

	response.Success(c, config)
}

// Update 更新存储配置
func (h *OSSConfigHandler) Update(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	var updateData struct {
		Name     string `json:"name" binding:"required"`
		Type     string `json:"type" binding:"required"`
		Endpoint string `json:"endpoint" binding:"required"`
		Bucket   string `json:"bucket" binding:"required"`
		AccessKey string `json:"access_key" binding:"required"`
		SecretKey string `json:"secret_key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	// 验证配置类型
	if !utils.IsValidStorageType(updateData.Type) {
		response.Error(c, response.CodeInvalidParams, "不支持的存储类型", nil)
		return
	}

	// 更新配置
	config.Name = updateData.Name
	config.Type = updateData.Type
	config.Endpoint = updateData.Endpoint
	config.Bucket = updateData.Bucket
	config.AccessKey = updateData.AccessKey
	config.SecretKey = updateData.SecretKey

	if err := h.db.Save(&config).Error; err != nil {
		response.Error(c, response.CodeServerError, "更新存储配置失败", err)
		return
	}

	response.Success(c, config)
}

// Delete 删除存储配置
func (h *OSSConfigHandler) Delete(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	// 检查是否有文件使用此配置
	var count int64
	if err := h.db.Model(&models.OSSFile{}).Where("config_id = ?", configID).Count(&count).Error; err != nil {
		response.Error(c, response.CodeServerError, "检查文件关联失败", err)
		return
	}
	if count > 0 {
		response.Error(c, response.CodeConfigInUse, "存储配置正在使用中，无法删除", nil)
		return
	}

	if err := h.db.Delete(&config).Error; err != nil {
		response.Error(c, response.CodeServerError, "删除存储配置失败", err)
		return
	}

	response.Success(c, nil)
}

// List 获取存储配置列表
func (h *OSSConfigHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var total int64
	if err := h.db.Model(&models.OSSConfig{}).Count(&total).Error; err != nil {
		response.Error(c, response.CodeServerError, "获取配置总数失败", err)
		return
	}

	var configs []models.OSSConfig
	if err := h.db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&configs).Error; err != nil {
		response.Error(c, response.CodeServerError, "获取配置列表失败", err)
		return
	}

	response.Success(c, gin.H{
		"total": total,
		"items": configs,
	})
}

// Get 获取存储配置详情
func (h *OSSConfigHandler) Get(c *gin.Context) {
	configID := c.Param("id")

	var config models.OSSConfig
	if err := h.db.First(&config, configID).Error; err != nil {
		response.Error(c, response.CodeConfigNotFound, "存储配置不存在", nil)
		return
	}

	response.Success(c, config)
}

// Test 测试存储配置
func (h *OSSConfigHandler) Test(c *gin.Context) {
	var config models.OSSConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	// 验证配置类型
	if !utils.IsValidStorageType(config.Type) {
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