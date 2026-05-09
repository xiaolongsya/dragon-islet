package handler

import (
	"dragon-islet/internal/global"
	"dragon-islet/pkg/deepseek"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)


type QuoteHandler struct{}

func (h *QuoteHandler) Get(c *gin.Context) {
	// 10% 概率触发隐藏情话彩蛋
	rand.Seed(time.Now().UnixNano())
	isSecret := rand.Float32() < 0.1

	var systemPrompt, userPrompt string
	if isSecret {
		systemPrompt = "你是一位古典诗人，擅长写含蓄而动人的情话。"
		userPrompt = `请生成一句含蓄优美的情话（不超过30字），然后用一句幽默接地气的大白话来解释它的真实含义。
返回格式（严格JSON）：{"quote":"...","explain":"...","type":"secret"}`
	} else {
		systemPrompt = "你是一位东方智者，擅长生成富含哲理的警句。"
		userPrompt = `请生成一句富有哲理的话（不超过30字，可以是原创或改编自名言），然后用一句幽默接地气的大白话来解释它的真实含义（可以适度诙谐调侃）。
返回格式（严格JSON）：{"quote":"...","explain":"...","type":"wisdom"}`
	}

	client := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	result, err := client.Chat([]deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"quote":   "山知道我，我知道你，就已足够。",
			"explain": "翻译成大白话就是：不需要全世界都懂我，有你就够了。",
			"type":    "wisdom",
		})
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(result))
}

