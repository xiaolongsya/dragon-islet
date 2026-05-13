package handler

import (
	"dragon-islet/internal/service"
	"net/http"
	"bufio"
	"encoding/json"
	"strings"
	"math/rand"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DragonHandler struct {
	dragonService service.DragonService
}

func NewDragonHandler(dragonSvc service.DragonService) *DragonHandler {
	return &DragonHandler{
		dragonService: dragonSvc,
	}
}

func (h *DragonHandler) GetStatus(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	dragon, err := h.dragonService.GetDragon(userID)
	if err != nil {
		items, _ := h.dragonService.GetItems(userID)
		c.JSON(http.StatusOK, gin.H{
			"has_dragon": false,
			"items":      items,
		})
		return
	}

	items, _ := h.dragonService.GetItems(userID)

	c.JSON(http.StatusOK, gin.H{
		"has_dragon": true,
		"dragon":     dragon,
		"items":      items,
	})
}

func (h *DragonHandler) Feed(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	event, err := h.dragonService.Feed(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "它吃得很开心",
		"event":   event,
	})
}

func (h *DragonHandler) Play(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	event, err := h.dragonService.Play(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "它发出了欢快的鸣叫",
		"event":   event,
	})
}

func (h *DragonHandler) Evolve(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	token := c.GetHeader("Authorization")
	msg, err := h.dragonService.Evolve(userID, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func (h *DragonHandler) Rename(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名字不能为空"})
		return
	}

	if err := h.dragonService.Rename(userID, req.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "名字已镌刻"})
}

func (h *DragonHandler) GenerateImage(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	token := c.GetHeader("Authorization") // 获取用户的 JWT 令牌
	
	if err := h.dragonService.GenerateImage(userID, token, ""); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "龙主正在凝聚灵力，请稍后查看"})
}

func (h *DragonHandler) Share(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	if err := h.dragonService.ShareToChat(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已将龙宝宝的真身分享至誓约广场"})
}

func (h *DragonHandler) GetTasks(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	tasks := h.dragonService.GetDailyTasks(userID)
	c.JSON(http.StatusOK, tasks)
}

func (h *DragonHandler) ClaimReward(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务标识"})
		return
	}

	msg, err := h.dragonService.ClaimTaskReward(userID, req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func (h *DragonHandler) UseItem(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要使用的道具"})
		return
	}

	msg, err := h.dragonService.UseItem(userID, req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func (h *DragonHandler) GetChatHistory(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	msgs, err := h.dragonService.GetDragonChatHistory(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func (h *DragonHandler) Chat(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "你想对它说什么？"})
		return
	}

	resp, dragon, err := h.dragonService.PrepareDragonChatStream(userID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	scanner := bufio.NewScanner(resp.Body)
	fullReply := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" { continue }
		if line == "data: [DONE]" { break }

		if strings.HasPrefix(line, "data: ") {
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &chunk)
			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				fullReply += content
				c.SSEvent("message", content)
				c.Writer.Flush()
			}
		}
	}

	h.dragonService.SaveDragonReply(userID, dragon.ID, fullReply)
	
	// 15% 概率触发随机奇遇
	if rand.Float32() < 0.15 {
		event, _ := h.dragonService.TriggerRandomEvent(userID, "chat")
		if event != "" {
			c.SSEvent("event", event)
			c.Writer.Flush()
		}
	}
}

func (h *DragonHandler) Release(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供密语以确认身份"})
		return
	}

	if err := h.dragonService.ReleaseDragon(userID, req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "契约已解除，它已回归星海"})
}

func (h *DragonHandler) GetSummary(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	dragon, _ := h.dragonService.GetDragon(userID)
	allDone := h.dragonService.CheckAllTasksCompleted(userID)
	
	c.JSON(http.StatusOK, gin.H{
		"has_dragon":       dragon != nil,
		"all_tasks_done": allDone,
	})
}
