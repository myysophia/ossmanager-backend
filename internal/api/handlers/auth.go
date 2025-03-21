package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/myysophia/ossmanager-backend/internal/auth"
	"github.com/myysophia/ossmanager-backend/internal/config"
	"github.com/myysophia/ossmanager-backend/internal/db"
	"github.com/myysophia/ossmanager-backend/internal/db/models"
	"github.com/myysophia/ossmanager-backend/internal/utils"
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
	*BaseHandler
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		BaseHandler: NewBaseHandler(),
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseError(c, utils.CodeInvalidParams, err)
		return
	}

	var user models.User
	if err := db.GetDB().Where("username = ?", req.Username).First(&user).Error; err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("用户名或密码错误"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("用户名或密码错误"))
		return
	}

	// 生成 JWT token
	jwtConfig := config.GetConfig().JWT
	token, err := auth.GenerateToken(&user, &jwtConfig)
	if err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("生成令牌失败"))
		return
	}

	utils.ResponseWithData(c, gin.H{
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
		utils.ResponseError(c, utils.CodeInvalidParams, errors.New("参数错误"))
		return
	}

	// 检查用户名是否已存在
	var count int64
	if err := db.GetDB().Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("检查用户名失败"))
		return
	}
	if count > 0 {
		utils.ResponseError(c, utils.CodeInvalidParams, errors.New("用户名已存在"))
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("密码加密失败"))
		return
	}

	// 创建用户
	user := models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
	}

	if err := db.GetDB().Create(&user).Error; err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("创建用户失败"))
		return
	}

	// 分配默认角色
	var defaultRole models.Role
	if err := db.GetDB().Where("name = ?", "user").First(&defaultRole).Error; err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("获取默认角色失败"))
		return
	}

	if err := db.GetDB().Model(&user).Association("Roles").Append(&defaultRole); err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("分配角色失败"))
		return
	}

	utils.ResponseWithData(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		utils.ResponseError(c, utils.CodeUnauthorized, errors.New("未登录"))
		return
	}

	var user models.User
	if err := db.GetDB().Preload("Roles").First(&user, userID).Error; err != nil {
		utils.ResponseError(c, utils.CodeNotFound, errors.New("用户不存在"))
		return
	}

	utils.ResponseWithData(c, gin.H{
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
		utils.ResponseError(c, utils.CodeInvalidParams, errors.New("参数错误"))
		return
	}

	// 从上下文获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		utils.ResponseError(c, utils.CodeUnauthorized, errors.New("未登录"))
		return
	}

	var user models.User
	if err := db.GetDB().First(&user, userID).Error; err != nil {
		utils.ResponseError(c, utils.CodeNotFound, errors.New("用户不存在"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		utils.ResponseError(c, utils.CodeInvalidParams, errors.New("原密码错误"))
		return
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("密码加密失败"))
		return
	}

	if err := db.GetDB().Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		utils.ResponseError(c, utils.CodeInternalError, errors.New("更新密码失败"))
		return
	}

	utils.ResponseWithData(c, nil)
}
