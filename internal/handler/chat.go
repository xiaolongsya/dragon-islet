package handler

import (
	"dragon-islet/internal/logic"
	"dragon-islet/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ChatHandler struct {
	chatService service.ChatService
}

func (h *ChatHandler) WsChat(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &logic.Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	logic.GlobalHub.Register(client)

	// 启动写协程
	go client.WritePump()
	// 启动读协程 (主要用于心跳)
	go client.ReadPump()
}

func (h *ChatHandler) Send(c *gin.Context) {
	// 获取当前登录用户 ID (从 JWT 中间件解析出来的)
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}
	userID := userIDVal.(uint)

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	if len([]rune(req.Content)) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "誓言过长，请精简至 500 字以内"})
		return
	}

	msg, willReply, err := h.chatService.SendMessage(userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "誓言已传达",
		"data":       msg,
		"will_reply": willReply,
	})
}

func (h *ChatHandler) List(c *gin.Context) {
	const pageSize = 15
	beforeIDStr := c.DefaultQuery("before_id", "0")
	beforeID, _ := strconv.ParseUint(beforeIDStr, 10, 64)

	messages, hasMore, err := h.chatService.GetMessages(pageSize, uint(beforeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":     messages,
		"has_more": hasMore,
	})
}

func (h *ChatHandler) Delete(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)

	if err := h.chatService.DeleteMessage(userID, uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "誓言已抹除"})
}

func (h *ChatHandler) MyMessages(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	messages, total, err := h.chatService.GetUserMessages(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  messages,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *ChatHandler) ForceReply(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uint)

	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if err := h.chatService.ForceAIReply(userID, req.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "秘宝已激活，请静候回响"})
}
