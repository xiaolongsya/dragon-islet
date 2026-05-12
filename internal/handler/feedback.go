package handler

import (
	"dragon-islet/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	feedbackService service.FeedbackService
}

func NewFeedbackHandler(feedbackSvc service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackSvc,
	}
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

func (h *FeedbackHandler) Delete(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	if err := h.feedbackService.DeleteFeedback(userID, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "信笺已抹除"})
}

func (h *FeedbackHandler) AdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	feedbacks, total, err := h.feedbackService.AdminGetList(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  feedbacks,
		"total": total,
	})
}

func (h *FeedbackHandler) AdminReply(c *gin.Context) {
	var req struct {
		ID      uint   `json:"id" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.feedbackService.AdminReply(req.ID, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "回复失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已传达龙主的回响"})
}
