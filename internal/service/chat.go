package service

import (
	"context"
	"dragon-islet/internal/global"
	"dragon-islet/internal/logic"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/deepseek"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type ChatService struct{}

// SendMessage 处理发送消息逻辑
func (s *ChatService) SendMessage(userID uint, content string) (*model.Message, bool, error) {
	// 1. 频率限制 (1分钟一次)
	ctx := context.Background()
	key := fmt.Sprintf("chat_limit:%d", userID)
	if exists, _ := global.REDIS.Exists(ctx, key).Result(); exists > 0 {
		return nil, false, errors.New("发言太频繁了，请稍后再试")
	}

	// 2. AI 内容安全检查 (DeepSeek)
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	checkPrompt := []deepseek.Message{
		{Role: "system", Content: "你是一个严厉的言论审判官。任何包含脏话、人身攻击，色情，暴力，辱骂的内容回复'REJECT'，否则回复'PASS'。"},
		{Role: "user", Content: content},
	}
	result, err := dsClient.Chat(checkPrompt)
	if err != nil || result == "REJECT" {
		return nil, false, errors.New("内容未通过龙语审核")
	}

	// 3. 掷骰子决定是否回复 (30% 概率) - 提前决定以便存入数据库
	rand.Seed(time.Now().UnixNano())
	willReply := rand.Float32() < 0.3

	// 4. 保存消息 (包含兴趣状态)
	msg := &model.Message{
		Content:    content,
		UserID:     &userID,
		AIInterest: willReply,
	}
	if err := global.DB.Create(msg).Error; err != nil {
		return nil, false, err
	}

	// 5. 预加载用户信息并广播
	global.DB.Preload("User").First(msg, msg.ID)
	logic.GlobalHub.BroadcastMessage(msg)

	// 6. 设置频率限制
	global.REDIS.Set(ctx, key, "1", 1*time.Minute)

	// 7. 如果感兴趣，触发异步回复
	if willReply {
		go s.TriggerAIReply(msg)
	}

	return msg, willReply, nil
}

// TriggerAIReply 触发异步回复逻辑 (移除概率判断，由外部控制)
func (s *ChatService) TriggerAIReply(userMsg *model.Message) {
	// 模拟思考延迟
	time.Sleep(2 * time.Second)

	// 获取最近 20 条上下文 (排除已撤回的消息)
	var contextMsgs []model.Message
	global.DB.Preload("User").Where("is_recalled = ?", false).Order("created_at desc").Limit(20).Find(&contextMsgs)
	
	// 倒序排列，让时间线正常
	for i, j := 0, len(contextMsgs)-1; i < j; i, j = i+1, j-1 {
		contextMsgs[i], contextMsgs[j] = contextMsgs[j], contextMsgs[i]
	}

	// 获取最近的一篇史诗作为大背景
	var lastArchive model.Archive
	global.DB.Order("date desc").First(&lastArchive)

	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))

	// 获取当前发言游侠信息
	var user model.User
	global.DB.First(&user, userMsg.UserID)

	// 构建上下文 Prompt
	aiPrompt := []deepseek.Message{
		{Role: "system", Content: fmt.Sprintf("你是一头名为'龙屿之主'的西方巨龙，性格高傲、睿智且带有一点幽默。你正在审视'龙屿'广场上的誓言。最近的岛屿史诗是《%s》。请根据最近的对话氛围，对当前游侠的发言进行简短精辟的点评。", lastArchive.Title)},
	}

	// 注入上下文
	for _, m := range contextMsgs {
		role := "user"
		name := "匿名游侠"
		if m.User.Username != "" { name = m.User.Username }
		content := m.Content
		if m.IsAIReply {
			role = "assistant"
			content = m.Content
		} else {
			content = fmt.Sprintf("[%s]: %s", name, m.Content)
		}
		aiPrompt = append(aiPrompt, deepseek.Message{Role: role, Content: content})
	}

	// 注入当前需要回复的消息
	aiPrompt = append(aiPrompt, deepseek.Message{Role: "user", Content: fmt.Sprintf("[%s]: %s", user.Username, userMsg.Content)})

	replyContent, err := dsClient.Chat(aiPrompt)
	if err != nil {
		fmt.Printf("AI 生成失败: %v\n", err)
		return
	}

	// 保存 AI 回复
	aiMsg := &model.Message{
		Content:          replyContent,
		IsAIReply:        true,
		ReplyToMessageID: &userMsg.ID,
		UserID:           nil, // AI 回复不属于任何用户，设为 nil 以存入 NULL
	}

	if err := global.DB.Create(aiMsg).Error; err != nil {
		fmt.Printf("AI 消息存入数据库失败: %v\n", err)
		return
	}

	// 广播 AI 回复
	global.DB.Preload("User").Preload("ReplyToMessage.User").First(aiMsg, aiMsg.ID)
	logic.GlobalHub.BroadcastMessage(aiMsg)
}

// GetMessages 游标分页获取聊天记录 (before_id=0 表示获取最新)
func (s *ChatService) GetMessages(limit int, beforeID uint) ([]model.Message, bool, error) {
	var messages []model.Message
	query := global.DB.Preload("User").Preload("ReplyToMessage").Order("created_at desc").Limit(limit + 1)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, false, err
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

// DeleteMessage 撤回消息 (仅限本人或管理员)
func (s *ChatService) DeleteMessage(userID uint, messageID uint) error {
	var msg model.Message
	if err := global.DB.Preload("User").First(&msg, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}

	if msg.IsRecalled {
		return errors.New("消息已撤回")
	}

	// 权限检查
	var user model.User
	global.DB.First(&user, userID)

	if msg.UserID == nil || (*msg.UserID != userID && user.Role != "admin") {
		return errors.New("无权撤回此消息")
	}

	// 执行撤回逻辑 (更新状态)
	msg.IsRecalled = true
	if err := global.DB.Save(&msg).Error; err != nil {
		return err
	}

	// 广播撤回状态 (复用原有广播逻辑，前端通过 ID 匹配并更新)
	logic.GlobalHub.BroadcastMessage(&msg)

	return nil
}

// GetUserMessages 获取特定用户的聊天记录 (分页)
func (s *ChatService) GetUserMessages(userID uint, limit int, offset int) ([]model.Message, int64, error) {
	var messages []model.Message
	var total int64

	db := global.DB.Model(&model.Message{}).Where("user_id = ?", userID)
	db.Count(&total)

	err := db.Preload("User").Preload("ReplyToMessage").
		Order("created_at desc").
		Limit(limit).Offset(offset).
		Find(&messages).Error

	return messages, total, err
}

// ForceAIReply 动用秘宝强制触发回复
func (s *ChatService) ForceAIReply(userID uint, messageID uint) error {
	ctx := context.Background()
	cooldownKey := fmt.Sprintf("force_reply_cooldown:%d", userID)
	
	// 1. 冷却检查
	if exists, _ := global.REDIS.Exists(ctx, cooldownKey).Result(); exists > 0 {
		ttl, _ := global.REDIS.TTL(ctx, cooldownKey).Result()
		return fmt.Errorf("秘宝冷却中，还需等待 %d 秒", int(ttl.Seconds()))
	}

	var msg model.Message
	if err := global.DB.First(&msg, messageID).Error; err != nil {
		return errors.New("该誓言已消散在虚空")
	}

	// 2. 规则检查
	if msg.UserID == nil || *msg.UserID != userID {
		return errors.New("你只能对自己被不屑的誓言使用秘宝")
	}
	if msg.AIInterest {
		return errors.New("龙主已给予关注，无需动用秘宝")
	}
	if msg.IsForceReplied {
		return errors.New("此誓言已动用过秘宝")
	}

	// 3. 状态更新与异步触发
	msg.IsForceReplied = true
	msg.AIInterest = true // 修改状态为感兴趣，触发回复
	if err := global.DB.Save(&msg).Error; err != nil {
		return err
	}

	// 广播状态更新
	global.DB.Preload("User").First(&msg, msg.ID)
	logic.GlobalHub.BroadcastMessage(&msg)

	// 设置5分钟冷却
	global.REDIS.Set(ctx, cooldownKey, "1", 5*time.Minute)

	go s.TriggerAIReply(&msg)

	return nil
}
