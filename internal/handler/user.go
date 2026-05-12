package handler

import (
	"dragon-islet/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	authService   service.AuthService
	userService   service.UserService
	dragonService service.DragonService
}

func NewUserHandler(authSvc service.AuthService, userSvc service.UserService, dragonSvc service.DragonService) *UserHandler {
	return &UserHandler{
		authService:   authSvc,
		userService:   userSvc,
		dragonService: dragonSvc,
	}
}

func (h *UserHandler) GetFortune(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	record, err := h.userService.GetDailyFortune(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "神龙祭司正忙，请稍后再求签"})
		return
	}

	// 推进每日修行：求签
	h.dragonService.UpdateTaskProgress(userID, "fortune", 1)
	
	// 显式构建返回数据，确保字段名绝对匹配前端
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"luck":           record.Luck,
			"verse":          record.Verse,
			"interpretation": record.Interpretation,
			"suit":           record.Suit,
			"avoid":          record.Avoid,
			"date":           record.Date,
		},
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Motto    string `json:"motto"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.authService.UpdateProfile(userID, req.Nickname, req.Avatar, req.Motto); err != nil {
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

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}
