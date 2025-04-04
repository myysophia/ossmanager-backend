package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/db"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/logger"
	"github.com/myysophia/ossmanager-backend/internal/utils"
	"go.uber.org/zap"
)

// RoleHandler 角色管理处理器
type RoleHandler struct {
	*BaseHandler
}

// NewRoleHandler 创建角色管理处理器
func NewRoleHandler() *RoleHandler {
	return &RoleHandler{
		BaseHandler: NewBaseHandler(),
	}
}

// List 获取角色列表
func (h *RoleHandler) List(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 构建查询
	query := db.GetDB().Model(&models.Role{})

	// 处理筛选条件
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Error("获取角色总数失败", zap.Error(err))
		h.InternalError(c, "获取角色总数失败")
		return
	}

	// 获取角色列表
	var roles []models.Role
	if err := query.Preload("Permissions").Offset((page - 1) * pageSize).Limit(pageSize).Find(&roles).Error; err != nil {
		logger.Error("获取角色列表失败", zap.Error(err))
		h.InternalError(c, "获取角色列表失败")
		return
	}

	h.Success(c, gin.H{
		"total": total,
		"items": roles,
	})
}

// Create 创建角色
func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required,min=2,max=32"`
		Description   string `json:"description"`
		PermissionIDs []uint `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "参数错误")
		return
	}

	// 检查角色名是否已存在
	var count int64
	if err := db.GetDB().Model(&models.Role{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		h.InternalError(c, "检查角色名失败")
		return
	}
	if count > 0 {
		h.Error(c, utils.CodeInvalidParams, "角色名已存在")
		return
	}

	// 创建角色
	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	// 开启事务
	tx := db.GetDB().Begin()

	if err := tx.Create(&role).Error; err != nil {
		tx.Rollback()
		h.InternalError(c, "创建角色失败")
		return
	}

	// 分配权限
	if len(req.PermissionIDs) > 0 {
		var permissions []models.Permission
		if err := tx.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err != nil {
			tx.Rollback()
			h.InternalError(c, "获取权限失败")
			return
		}

		if err := tx.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			tx.Rollback()
			h.InternalError(c, "分配权限失败")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		h.InternalError(c, "提交事务失败")
		return
	}

	// 记录审计日志
	h.createAuditLog(c, "CREATE", "ROLE", strconv.FormatUint(uint64(role.ID), 10), "创建角色")

	h.Success(c, role)
}

// Get 获取角色详情
func (h *RoleHandler) Get(c *gin.Context) {
	roleID := c.Param("id")

	var role models.Role
	if err := db.GetDB().Preload("Permissions").First(&role, roleID).Error; err != nil {
		h.NotFound(c, "角色不存在")
		return
	}

	h.Success(c, role)
}

// Update 更新角色
func (h *RoleHandler) Update(c *gin.Context) {
	roleID := c.Param("id")

	var role models.Role
	if err := db.GetDB().First(&role, roleID).Error; err != nil {
		h.NotFound(c, "角色不存在")
		return
	}

	var req struct {
		Name          string `json:"name" binding:"required,min=2,max=32"`
		Description   string `json:"description"`
		PermissionIDs []uint `json:"permission_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "参数错误")
		return
	}

	// 检查角色名是否已被其他角色使用
	var count int64
	if err := db.GetDB().Model(&models.Role{}).Where("name = ? AND id != ?", req.Name, roleID).Count(&count).Error; err != nil {
		h.InternalError(c, "检查角色名失败")
		return
	}
	if count > 0 {
		h.Error(c, utils.CodeInvalidParams, "角色名已存在")
		return
	}

	// 开启事务
	tx := db.GetDB().Begin()

	// 更新基本信息
	if err := tx.Model(&role).Updates(models.Role{
		Name:        req.Name,
		Description: req.Description,
	}).Error; err != nil {
		tx.Rollback()
		h.InternalError(c, "更新角色失败")
		return
	}

	// 更新权限
	if len(req.PermissionIDs) > 0 {
		var permissions []models.Permission
		if err := tx.Where("id IN ?", req.PermissionIDs).Find(&permissions).Error; err != nil {
			tx.Rollback()
			h.InternalError(c, "获取权限失败")
			return
		}

		if err := tx.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			tx.Rollback()
			h.InternalError(c, "更新权限失败")
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		h.InternalError(c, "提交事务失败")
		return
	}

	// 记录审计日志
	h.createAuditLog(c, "UPDATE", "ROLE", roleID, "更新角色信息")

	// 重新获取角色信息（包含权限）
	if err := db.GetDB().Preload("Permissions").First(&role, roleID).Error; err != nil {
		h.NotFound(c, "获取更新后的角色信息失败")
		return
	}

	h.Success(c, role)
}

// Delete 删除角色
func (h *RoleHandler) Delete(c *gin.Context) {
	roleID := c.Param("id")

	var role models.Role
	if err := db.GetDB().First(&role, roleID).Error; err != nil {
		h.NotFound(c, "角色不存在")
		return
	}

	// 开启事务
	tx := db.GetDB().Begin()

	// 清除角色权限关联
	if err := tx.Model(&role).Association("Permissions").Clear(); err != nil {
		tx.Rollback()
		h.InternalError(c, "清除角色权限关联失败")
		return
	}

	// 清除用户角色关联
	if err := tx.Model(&role).Association("Users").Clear(); err != nil {
		tx.Rollback()
		h.InternalError(c, "清除用户角色关联失败")
		return
	}

	// 删除角色
	if err := tx.Delete(&role).Error; err != nil {
		tx.Rollback()
		h.InternalError(c, "删除角色失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		h.InternalError(c, "提交事务失败")
		return
	}

	// 记录审计日志
	h.createAuditLog(c, "DELETE", "ROLE", roleID, "删除角色")

	h.Success(c, nil)
}

// createAuditLog 创建审计日志
func (h *RoleHandler) createAuditLog(c *gin.Context, action, resourceType, resourceID, details string) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")

	auditLog := models.AuditLog{
		UserID:       userID.(uint),
		Username:     username.(string),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Status:       "SUCCESS",
	}

	if err := db.GetDB().Create(&auditLog).Error; err != nil {
		logger.Error("创建审计日志失败",
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID),
			zap.Error(err))
	}
}
