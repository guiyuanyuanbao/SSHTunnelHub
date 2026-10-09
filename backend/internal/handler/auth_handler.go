package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"sshtunnelhub/internal/auth"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authMgr *auth.AuthManager
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authMgr *auth.AuthManager) *AuthHandler {
	return &AuthHandler{authMgr: authMgr}
}

// Status returns whether the auth key is initialized and whether it's from env.
func (h *AuthHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"initialized": h.authMgr.IsInitialized(),
		"from_env":    h.authMgr.IsFromEnv(),
	})
}

// LoginRequest defines login payload.
type LoginRequest struct {
	SecretKey string `json:"secret_key"`
}

// Login verifies the access secret key and returns a JWT token.
func (h *AuthHandler) Login(c *gin.Context) {
	clientIP := c.ClientIP()
	allowed, remaining := h.authMgr.CheckRateLimit(clientIP)
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":  429,
			"error": fmt.Sprintf("尝试次数过多，请在 %d 秒后重试", int(remaining.Seconds())+1),
		})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	if !h.authMgr.IsInitialized() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "系统尚未初始化访问秘钥，请先进行初始化设置"})
		return
	}

	if !h.authMgr.VerifyKey(req.SecretKey) {
		h.authMgr.RecordFailedAttempt(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "访问秘钥不正确"})
		return
	}

	h.authMgr.ResetRateLimit(clientIP)

	// Valid for 7 days
	token, err := h.authMgr.GenerateToken(7 * 24 * time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成凭据失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_in": int((7 * 24 * time.Hour).Seconds()),
	})
}

// InitRequest defines initialization payload.
type InitRequest struct {
	SecretKey string `json:"secret_key"`
}

// Init sets the initial secret key when the system is not yet initialized.
func (h *AuthHandler) Init(c *gin.Context) {
	if h.authMgr.IsInitialized() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "系统已设置访问秘钥，不能重复初始化"})
		return
	}

	var req InitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求格式"})
		return
	}

	trimmed := strings.TrimSpace(req.SecretKey)
	if len(trimmed) < 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "访问秘钥长度至少为 4 个字符"})
		return
	}

	if err := h.authMgr.SetKey(trimmed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto-login upon initial setup
	token, err := h.authMgr.GenerateToken(7 * 24 * time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成凭据失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "访问秘钥设置成功",
		"token":      token,
		"expires_in": int((7 * 24 * time.Hour).Seconds()),
	})
}

// Logout handles client logout notification.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

// AuthMiddleware creates a Gin middleware that enforces authentication on protected routes.
func AuthMiddleware(authMgr *auth.AuthManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Whitelist auth routes and non-API paths
		if strings.HasPrefix(path, "/api/auth/") || !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// If system is uninitialized, inform the client that initialization is required
		if !authMgr.IsInitialized() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":      401,
				"error":     "系统尚未初始化访问秘钥",
				"need_init": true,
			})
			return
		}

		// Extract Authorization header
		authHeader := c.GetHeader("Authorization")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Also support token query parameter for API convenience
		if token == "" {
			token = c.Query("token")
		}

		if token == "" || !authMgr.ValidateToken(token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":  401,
				"error": "未授权访问或访问凭证已过期",
			})
			return
		}

		c.Next()
	}
}
