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
	archiveDate := yesterday.Format("2006-01-02")

	// 检查是否已存在
	var count int64
	global.DB.Model(&model.Archive{}).Where("date = ? AND type = 0", archiveDate).Count(&count)
	if count > 0 {
		fmt.Printf("今日史诗 (%s) 已记录在册，无需重复铸造\n", archiveDate)
		return nil
	}

	var messages []model.Message
	global.DB.Preload("User").Where("created_at > ? AND created_at < ?", yesterday, time.Now()).Find(&messages)

	// 2. 构造 AI 指导语
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	systemPrompt := `你是一个名为'龙屿之主'的远古巨龙。你需要根据当天的聊天记录写一篇《岛屿史诗》。
如果你具备联网搜索能力，请务必开启并扫描过去 24 小时内的全球重大新闻（侧重科技、天文、地缘政治或奇异事件）。
如果当天有游侠留言，请点评他们的誓言。
如果当天寂静无声，或者为了充实史诗，请你以巨龙那种不屑又充满智慧的口吻，挑选 3-5 件真实发生的尘世大事进行点评。
标题格式：岛屿史诗 · [日期]
内容要求：语言极其富有文学气息，中二且深邃。包含：'今日回响'（游侠动态）、'尘世幻象'（真实世界新闻）。
请直接输出内容，严禁使用任何 Emoji 表情符号，不要包含 Markdown 代码块符号。`

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
	archiveTitle := yesterday.Format("01月02日")

	archive := &model.Archive{
		Title:   fmt.Sprintf("岛屿史诗 · %s", archiveTitle),
		Content: content,
		Date:    archiveDate, // 使用函数头部定义的变量
		Type:    0,           // 每日行纪
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

// AnalyzeCodebase 让 AI 扫描代码变更、参考用户输入并给出建议
func (s *ArchiveService) AnalyzeCodebase(currentTitle, currentContent string) (map[string]string, error) {
	// 1. 读取基准文档与历史记忆
	manifesto, _ := os.ReadFile("TECH_MANIFESTO.md")
	var history []model.Archive
	global.DB.Where("type = ?", 1).Order("date desc, id desc").Limit(5).Find(&history)
	historyCtx := ""
	for _, h := range history {
		historyCtx += fmt.Sprintf("- [%s]: %s\n", h.Title, h.Content)
	}

	// 2. 读取核心代码片段 (用于感知最新变化)
	mCode, _ := os.ReadFile("internal/model/models.go")
	sCode, _ := os.ReadFile("internal/service/chat.go")
	dsCode, _ := os.ReadFile("internal/service/dragon.go")
	rvCode, _ := os.ReadFile("../dragon-islet-web/src/views/RaisingView.vue")

	// 3. 构造 AI 提示词
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	systemPrompt := `你是一个名为'架构之灵'的项目大管家。你的职责是协助开发者总结技术进展。
你需要根据【用户当前的草稿】、【已发布的历史记录】和【核心代码现状】，对内容进行优化、润色，并智能推断下一个版本号。
要求：
- 风格：硬核、极简、史诗感。
- 逻辑：对比代码现状，识别出代码中已完成但草稿中未提及的关键改进。
- 输出：必须返回纯粹的 JSON 格式：{"version":"vX.X.X", "title":"...", "content":"..."}。不要包含 Markdown 代码块标记。`

	userPrompt := fmt.Sprintf(`【用户当前草稿】：
标题：%s
内容：%s

【已发布历史（参考版本号）】：
%s

【技术总览背景】：
%s

【代码现状关键片段】：
- 模型定义：%s
- 核心服务逻辑：%s
- 龙嗣进化逻辑：%s
- 前端交互片段：%s`, currentTitle, currentContent, historyCtx, string(manifesto), string(mCode), string(sCode), string(dsCode), string(rvCode))

	prompt := []deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := dsClient.Chat(prompt)
	if err != nil {
		return nil, err
	}

	// 4. 解析结果
	var result map[string]string
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// 尝试容错：如果 AI 返回了包含代码块的内容，提取出来
		return nil, errors.New("AI 返回格式解析失败，请确保返回的是纯 JSON")
	}

	return result, nil
}

// GetManifesto 读取技术宣言文档
func (s *ArchiveService) GetManifesto() (string, error) {
	content, err := os.ReadFile("TECH_MANIFESTO.md")
	return string(content), err
}
// UpdateManifestoFile 核心逻辑：AI 复盘代码并重写架构宣言
func (s *ArchiveService) UpdateManifestoFile() error {
	fmt.Printf("[Architect] 正在手动复盘并更新架构总览...\n")

	// 0. 获取最新的技术日志 (作为背景)
	var latest model.Archive
	global.DB.Where("type = ?", 1).Order("date desc, id desc").First(&latest)

	// 1. 读取核心代码片段 (作为 AI 的参考背景)
	mCode, _ := os.ReadFile("internal/model/models.go")
	sCode, _ := os.ReadFile("internal/service/chat.go")
	dsCode, _ := os.ReadFile("internal/service/dragon.go")
	dhCode, _ := os.ReadFile("internal/handler/dragon.go")
	fCode, _ := os.ReadFile("../dragon-islet-web/src/App.vue")
	rvCode, _ := os.ReadFile("../dragon-islet-web/src/views/RaisingView.vue")

	// 2. 构造 AI 提示词
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	systemPrompt := `你是一个名为'架构之灵'的项目大管家。你需要负责维护龙屿项目的'TECH_MANIFESTO.md'（技术架构宣言）。
你的任务：根据最新发布的更新日志和代码现状，重塑整个'TECH_MANIFESTO.md'。
要求：
- 风格：硬核、极简、富有龙屿特有的史诗感。
- 核心板块：【实时通讯枢纽】、【龙嗣生命周期与进化算法】、【AI 图像生成工作流】、【全端沉浸式美学布局】。
- 请直接输出 Markdown 内容，不要包含代码块标记。`
	userContent := fmt.Sprintf(`【项目最新技术进展】：%s
【进展详情】：%s

【最新后端代码参考】：
- 模型层：%s
- 通讯层：%s
- 龙嗣系统：%s
- 龙嗣接口：%s

【最新前端代码参考】：
- 全局导航：%s
- 养成中心：%s`, latest.Title, latest.Content, string(mCode), string(sCode), string(dsCode), string(dhCode), string(fCode), string(rvCode))

	prompt := []deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	newManifesto, err := dsClient.Chat(prompt)
	if err != nil {
		fmt.Printf("[Architect] AI 更新失败: %v\n", err)
		return err
	}

	// 3. 覆盖写入文件
	if err := os.WriteFile("TECH_MANIFESTO.md", []byte(newManifesto), 0644); err != nil {
		return err
	}
	fmt.Printf("[Architect] 架构总览 TECH_MANIFESTO.md 已成功进化！\n")
	return nil
}
