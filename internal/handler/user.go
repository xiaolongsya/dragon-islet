package handler

import (
	"dragon-islet/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	authService service.AuthService
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.authService.UpdateProfile(userID, req.Nickname, req.Avatar); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "契约已更新"})
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Code     string `json:"code" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供验证码和新密语"})
		return
	}

	if err := h.authService.UpdatePasswordInternal(userID, req.Code, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密语重塑成功"})
}
