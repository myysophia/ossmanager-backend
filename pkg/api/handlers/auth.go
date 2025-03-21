package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ninesun/ossmanager-backend/pkg/db"
	"github.com/ninesun/ossmanager-backend/pkg/jwt"
	"github.com/ninesun/ossmanager-backend/pkg/models"
	"github.com/ninesun/ossmanager-backend/pkg/response"
	"github.com/ninesun/ossmanager-backend/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Email    string `json:"email" binding:"required,email"`
}

// AuthHandler 认证处理器
type AuthHandler struct {
	db *db.DB
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *db.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	var user models.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Error(c, response.CodeUserNotFound, "用户不存在", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response.Error(c, response.CodeInvalidPassword, "密码错误", nil)
		return
	}

	// 生成 JWT token
	token, err := jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		response.Error(c, response.CodeServerError, "生成token失败", err)
		return
	}

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"roles":    user.Roles,
		},
	})
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=32"`
		Password string `json:"password" binding:"required,min=6,max=32"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	// 检查用户名是否已存在
	var count int64
	if err := h.db.Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		response.Error(c, response.CodeServerError, "检查用户名失败", err)
		return
	}
	if count > 0 {
		response.Error(c, response.CodeUserExists, "用户名已存在", nil)
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, response.CodeServerError, "密码加密失败", err)
		return
	}

	// 创建用户
	user := models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
	}

	if err := h.db.Create(&user).Error; err != nil {
		response.Error(c, response.CodeServerError, "创建用户失败", err)
		return
	}

	// 分配默认角色
	var defaultRole models.Role
	if err := h.db.Where("name = ?", "user").First(&defaultRole).Error; err != nil {
		response.Error(c, response.CodeServerError, "获取默认角色失败", err)
		return
	}

	if err := h.db.Model(&user).Association("Roles").Append(&defaultRole); err != nil {
		response.Error(c, response.CodeServerError, "分配角色失败", err)
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// GetUserInfo 获取用户信息
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID := utils.GetUserID(c)

	var user models.User
	if err := h.db.Preload("Roles").First(&user, userID).Error; err != nil {
		response.Error(c, response.CodeUserNotFound, "用户不存在", nil)
		return
	}

	response.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"roles":    user.Roles,
	})
}

// UpdatePassword 更新密码
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, response.CodeInvalidParams, "参数错误", err)
		return
	}

	userID := utils.GetUserID(c)

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response.Error(c, response.CodeUserNotFound, "用户不存在", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		response.Error(c, response.CodeInvalidPassword, "原密码错误", nil)
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.Error(c, response.CodeServerError, "密码加密失败", err)
		return
	}

	if err := h.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		response.Error(c, response.CodeServerError, "更新密码失败", err)
		return
	}

	response.Success(c, nil)
} 