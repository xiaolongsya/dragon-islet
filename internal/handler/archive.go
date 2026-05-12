package handler

import (
	"dragon-islet/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ArchiveHandler struct {
	archiveService service.ArchiveService
}

func (h *ArchiveHandler) List(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	typeStr := c.DefaultQuery("type", "0") // 默认查询行纪
	limit, _ := strconv.Atoi(limitStr)
	archiveType, _ := strconv.Atoi(typeStr)

	list, err := h.archiveService.GetArchiveListByType(limit, archiveType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取史诗失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// Create 手动发布史诗 (主要用于技术进展)
func (h *ArchiveHandler) Create(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Type    int    `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不足"})
		return
	}

	if err := h.archiveService.CreateArchive(req.Title, req.Content, req.Type); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "铸龙图谱已更新"})
}

// ManualGenerate 供管理员手动触发生成的接口 (调试用)
func (h *ArchiveHandler) ManualGenerate(c *gin.Context) {
	if err := h.archiveService.GenerateDailyArchive(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "史诗已成"})
}

// Analyze 唤醒架构师进行代码分析
func (h *ArchiveHandler) Analyze(c *gin.Context) {
	suggestion, err := h.archiveService.AnalyzeCodebase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, suggestion)
}

// GetManifesto 获取技术总览文档
func (h *ArchiveHandler) GetManifesto(c *gin.Context) {
	content, err := h.archiveService.GetManifesto()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取宣言失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content})
}
