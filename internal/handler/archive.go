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
	limit, _ := strconv.Atoi(limitStr)

	list, err := h.archiveService.GetArchiveList(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取史诗失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// ManualGenerate 供管理员手动触发生成的接口 (调试用)
func (h *ArchiveHandler) ManualGenerate(c *gin.Context) {
	if err := h.archiveService.GenerateDailyArchive(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "史诗已成"})
}
