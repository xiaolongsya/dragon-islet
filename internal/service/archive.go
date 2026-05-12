package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/deepseek"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
			if m.User.Username != "" {
				name = m.User.Username
			}
			if m.IsAIReply {
				name = "龙屿之主"
			}
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
	archiveDate := yesterday.Format("2006-01-02")
	archiveTitle := yesterday.Format("01月02日")

	archive := &model.Archive{
		Title:   fmt.Sprintf("岛屿史诗 · %s", archiveTitle),
		Content: content,
		Date:    archiveDate,
		Type:    0, // 每日行纪
	}

	return global.DB.Create(archive).Error
}

// GetArchiveListByType 获取特定类型的史诗列表
func (s *ArchiveService) GetArchiveListByType(limit int, archiveType int) ([]model.Archive, error) {
	var archives []model.Archive
	err := global.DB.Where("type = ?", archiveType).Order("date desc, id desc").Limit(limit).Find(&archives).Error
	return archives, err
}

// CreateArchive 手动创建史诗记录
func (s *ArchiveService) CreateArchive(title, content string, archiveType int) error {
	archive := &model.Archive{
		Title:   title,
		Content: content,
		Date:    time.Now().Format("2006-01-02"),
		Type:    archiveType,
	}
	return global.DB.Create(archive).Error
}

// AnalyzeCodebase 让 AI 扫描代码变更并给出建议
func (s *ArchiveService) AnalyzeCodebase() (map[string]string, error) {
	// 1. 读取基准文档
	manifesto, _ := os.ReadFile("TECH_MANIFESTO.md")

	// 2. 读取核心代码片段 (后端)
	modelsCode, _ := os.ReadFile("internal/model/models.go")
	chatServiceCode, _ := os.ReadFile("internal/service/chat.go")

	// 尝试读取前端代码 (假设在同级目录)
	frontendApp, _ := os.ReadFile("../dragon-islet-web/src/App.vue")

	// 3. 构造 AI 提示词
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	prompt := []deepseek.Message{
		{Role: "system", Content: "你是一个资深全栈架构师，正在负责'龙屿'项目的演进。你需要对比现有的技术宣言和当前代码片段，识别出最新的变更，并建议下一个版本号、标题和技术摘要内容。请务必返回纯粹的 JSON 格式，不要包含 Markdown 代码块符号，格式如下：{\"version\":\"...\", \"title\":\"...\", \"content\":\"...\"}。内容要求硬核且干练。"},
		{Role: "user", Content: fmt.Sprintf("【技术宣言】:\n%s\n\n【后端模型】:\n%s\n\n【核心逻辑】:\n%s\n\n【前端核心】:\n%s", string(manifesto), string(modelsCode), string(chatServiceCode), string(frontendApp))},
	}

	resp, err := dsClient.Chat(prompt)
	if err != nil {
		return nil, err
	}

	// 4. 解析结果
	var result map[string]string
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, errors.New("AI 返回格式解析失败，请确保返回的是纯 JSON")
	}

	return result, nil
}

// GetManifesto 读取技术宣言文档
func (s *ArchiveService) GetManifesto() (string, error) {
	content, err := os.ReadFile("TECH_MANIFESTO.md")
	return string(content), err
}
