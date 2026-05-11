package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/deepseek"
	"fmt"
	"time"
)

type ArchiveService struct{}

// GenerateDailyArchive 生成每日史诗
func (s *ArchiveService) GenerateDailyArchive() error {
	// 1. 获取过去 24 小时的消息
	yesterday := time.Now().Add(-24 * time.Hour)
	var messages []model.Message
	global.DB.Preload("User").Where("created_at > ?", yesterday).Find(&messages)

	// 2. 构造 AI 指导语
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	systemPrompt := `你是一个名为'龙屿之主'的远古巨龙。你需要根据当天的聊天记录写一篇《岛屿史诗》。
如果你具备联网搜索能力，请务必开启并扫描过去 24 小时内的全球重大新闻（侧重科技、天文、地缘政治或奇异事件）。
如果当天有游侠留言，请点评他们的誓言。
如果当天寂静无声，或者为了充实史诗，请你以巨龙那种不屑又充满智慧的口吻，挑选 3-5 件真实发生的尘世大事进行点评。
标题格式：岛屿史诗 · [日期]
内容要求：语言极其富有文学气息，中二且深邃。包含：'今日回响'（游侠动态）、'尘世幻象'（真实世界新闻）。
请直接输出内容，不要包含 Markdown 代码块符号。`

	userContent := ""
	if len(messages) == 0 {
		userContent = fmt.Sprintf("今日岛屿寂静。现在是 %s，请龙主俯瞰凡间，聊聊尘世间的最新变动，并降下神谕。", time.Now().Format("2006-01-02"))
	} else {
		chatContext := ""
		for _, m := range messages {
			name := "匿名游侠"
			if m.User.Username != "" { name = m.User.Username }
			if m.IsAIReply { name = "龙屿之主" }
			chatContext += fmt.Sprintf("[%s]: %s\n", name, m.Content)
		}
		userContent = fmt.Sprintf("今日日期：%s\n游侠们的对话记录如下：\n%s\n请龙主总结并侃侃而谈。", time.Now().Format("2006-01-02"), chatContext)
	}

	prompt := []deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	content, err := dsClient.Chat(prompt)
	if err != nil {
		return err
	}

	// 3. 保存
	archive := &model.Archive{
		Title:   fmt.Sprintf("岛屿史诗 · %s", time.Now().Format("01月02日")),
		Content: content,
		Date:    time.Now().Format("2006-01-02"),
	}

	return global.DB.Create(archive).Error
}

// GetArchiveList 获取史诗列表
func (s *ArchiveService) GetArchiveList(limit int) ([]model.Archive, error) {
	var archives []model.Archive
	err := global.DB.Order("date desc").Limit(limit).Find(&archives).Error
	return archives, err
}
