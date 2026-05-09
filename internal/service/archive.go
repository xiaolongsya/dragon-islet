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

	if len(messages) == 0 {
		return nil // 没人说话，不写史诗
	}

	// 2. 格式化对话给 AI
	chatContext := ""
	for _, m := range messages {
		name := "匿名游侠"
		if m.User.Username != "" {
			name = m.User.Username
		}
		if m.IsAIReply {
			name = "龙屿之主"
		}
		chatContext += fmt.Sprintf("[%s]: %s\n", name, m.Content)
	}

	// 3. 调用 AI 创作
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	prompt := []deepseek.Message{
		{Role: "system", Content: "你是一个名为'龙屿观察者'的高级史官。你需要根据当天的聊天记录，写一篇富有文学气息、中二且有趣的《龙屿史诗》。包含：今日头条、英雄榜单、精彩回响。请直接输出内容，不要包含 Markdown 代码块符号。"},
		{Role: "user", Content: "今日对话记录：\n" + chatContext},
	}

	content, err := dsClient.Chat(prompt)
	if err != nil {
		return err
	}

	// 4. 保存
	archive := &model.Archive{
		Title:   fmt.Sprintf("龙屿史诗：纪元 %s", time.Now().Format("2006-01-02")),
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
