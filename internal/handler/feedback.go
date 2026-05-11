package handler

import (
	"dragon-islet/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	feedbackService service.FeedbackService
}

func (h *FeedbackHandler) Submit(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	if err := h.feedbackService.CreateFeedback(userID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "匿名信已传达"})
}

func (h *FeedbackHandler) List(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	feedbacks, err := h.feedbackService.GetUserFeedbacks(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": feedbacks})
}
