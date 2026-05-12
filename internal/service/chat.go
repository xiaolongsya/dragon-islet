package service

import (
	"context"
	"dragon-islet/internal/global"
	"dragon-islet/internal/logic"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/apimart"
	"dragon-islet/pkg/deepseek"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type ChatService struct{}

// SendMessage 处理发送消息逻辑
func (s *ChatService) SendMessage(userID uint, content string) (*model.Message, bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("chat_limit:%d", userID)
	if exists, _ := global.REDIS.Exists(ctx, key).Result(); exists > 0 {
		return nil, false, errors.New("发言太频繁了，请稍后再试")
	}

	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	checkPrompt := []deepseek.Message{
		{Role: "system", Content: "你是一个严厉的言论审判官。任何包含脏话、人身攻击，色情，暴力，辱骂的内容回复'REJECT'，否则回复'PASS'。"},
		{Role: "user", Content: content},
	}
	result, err := dsClient.Chat(checkPrompt)
	if err != nil || result == "REJECT" {
		return nil, false, errors.New("内容未通过龙语审核")
	}

	rand.Seed(time.Now().UnixNano())
	willReply := rand.Float32() < 0.3

	msg := &model.Message{
		Content:    content,
		UserID:     &userID,
		AIInterest: willReply,
	}
	if err := global.DB.Create(msg).Error; err != nil {
		return nil, false, err
	}

	global.DB.Preload("User").First(msg, msg.ID)
	logic.GlobalHub.BroadcastMessage(msg)

	global.REDIS.Set(ctx, key, "1", 1*time.Minute)

	if rand.Float32() < 0.1 {
		var userItem model.UserItem
		if err := global.DB.Where("user_id = ? AND type = ?", userID, "toy").First(&userItem).Error; err != nil {
			userItem = model.UserItem{UserID: userID, Type: "toy", Count: 1}
			global.DB.Create(&userItem)
		} else {
			userItem.Count++
			global.DB.Save(&userItem)
		}
	}

	go s.GainExperience(userID)

	// 更新每日交流任务
	ds := &DragonService{}
	ds.UpdateTaskProgress(userID, "chat", 1)

	if willReply {
		go s.TriggerAIReply(msg)
	}

	return msg, willReply, nil
}

// TriggerAIReply 触发异步回复逻辑
func (s *ChatService) TriggerAIReply(userMsg *model.Message) {
	time.Sleep(2 * time.Second)

	var contextMsgs []model.Message
	global.DB.Preload("User").Where("is_recalled = ?", false).Order("created_at desc").Limit(20).Find(&contextMsgs)
	
	for i, j := 0, len(contextMsgs)-1; i < j; i, j = i+1, j-1 {
		contextMsgs[i], contextMsgs[j] = contextMsgs[j], contextMsgs[i]
	}

	var lastArchive model.Archive
	global.DB.Order("date desc").First(&lastArchive)

	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))

	var user model.User
	global.DB.First(&user, userMsg.UserID)

	systemPrompt := fmt.Sprintf(`你是一头名为'龙屿之主'的西方巨龙，性格高傲、睿智且带有一点幽默。你正在审视'龙屿'广场上的誓言。最近的岛屿史诗是《%s》。

【核心规则：龙蛋赐予】
如果你游侠向你索要龙蛋、龙嗣或表达养龙愿望，请根据你的心情决定。
1. 如果你决定赐予：请先进行感性的回复，并在回复内容的最后一行，务必严格按照以下格式输出指令块：
操作：给予龙蛋
名字：[你为这头龙起的真名]
给予用户：[该游侠的名字]

2. 如果你不予赐予：请直接嘲讽或拒绝，回复中绝对不要出现上述指令块。
注：龙蛋虽然珍贵，但作为龙主，你应该慷慨一些。只要游侠诚心祈求，建议赐予概率保持在 60% 以上。`, lastArchive.Title)

	aiPrompt := []deepseek.Message{
		{Role: "system", Content: systemPrompt},
	}

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

	aiPrompt = append(aiPrompt, deepseek.Message{Role: "user", Content: fmt.Sprintf("[%s]: %s", user.Username, userMsg.Content)})

	replyContent, err := dsClient.Chat(aiPrompt)
	if err != nil {
		return
	}

	hasEggDirective := strings.Contains(replyContent, "操作：给予龙蛋")
	dragonName := "无名之蛋"
	
	if hasEggDirective {
		lines := strings.Split(replyContent, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "名字：") {
				dragonName = strings.TrimSpace(strings.TrimPrefix(line, "名字："))
				break
			}
		}
	}
	
	replyContent = strings.TrimSpace(replyContent)

	aiMsg := &model.Message{
		Content:          replyContent,
		IsAIReply:        true,
		ReplyToMessageID: &userMsg.ID,
	}

	if err := global.DB.Create(aiMsg).Error; err != nil {
		return
	}

	global.DB.Preload("User").Preload("ReplyToMessage.User").First(aiMsg, aiMsg.ID)
	logic.GlobalHub.BroadcastMessage(aiMsg)

	if hasEggDirective {
		s.DeliverEgg(userMsg, dragonName)
	}
}

// DeliverEgg 发放龙蛋
func (s *ChatService) DeliverEgg(userMsg *model.Message, name string) {
	var existingDragon model.Dragon
	if err := global.DB.Where("user_id = ? AND is_gone = ?", userMsg.UserID, false).First(&existingDragon).Error; err == nil {
		return
	}

	rarity := "common"
	r := rand.Float32()
	if r < 0.05 {
		rarity = "epic"
	} else if r < 0.35 {
		rarity = "rare"
	}

	now := time.Now()
	newDragon := &model.Dragon{
		UserID:       *userMsg.UserID,
		Name:         name,
		Rarity:       rarity,
		Stage:        0, 
		Exp:          0,
		LastFedAt:    &now,
		LastPlayedAt: &now,
		BasePrompt:   "A mystical dragon egg, " + rarity + " style, glowing runes, cinematic lighting",
		Seed:         rand.Int63(),
	}
	global.DB.Create(newDragon)
}

// GetMessages 获取消息
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

// DeleteMessage 撤回
func (s *ChatService) DeleteMessage(userID uint, messageID uint) error {
	var msg model.Message
	if err := global.DB.Preload("User").First(&msg, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}

	if msg.IsRecalled {
		return errors.New("消息已撤回")
	}

	var user model.User
	global.DB.First(&user, userID)

	if msg.UserID == nil || (*msg.UserID != userID && user.Role != "admin") {
		return errors.New("无权撤回此消息")
	}

	msg.IsRecalled = true
	global.DB.Save(&msg)
	logic.GlobalHub.BroadcastMessage(&msg)
	return nil
}

// ForceAIReply 秘宝触发
func (s *ChatService) ForceAIReply(userID uint, messageID uint) error {
	ctx := context.Background()
	cooldownKey := fmt.Sprintf("force_reply_cooldown:%d", userID)
	
	if exists, _ := global.REDIS.Exists(ctx, cooldownKey).Result(); exists > 0 {
		ttl, _ := global.REDIS.TTL(ctx, cooldownKey).Result()
		return fmt.Errorf("秘宝冷却中，还需等待 %d 秒", int(ttl.Seconds()))
	}

	var msg model.Message
	if err := global.DB.First(&msg, messageID).Error; err != nil {
		return errors.New("誓言已消散")
	}

	msg.IsForceReplied = true
	msg.AIInterest = true 
	global.DB.Save(&msg)

	global.DB.Preload("User").First(&msg, msg.ID)
	logic.GlobalHub.BroadcastMessage(&msg)

	global.REDIS.Set(ctx, cooldownKey, "1", 1*time.Minute)
	go s.TriggerAIReply(&msg)
	return nil
}

// GainExperience 增加用户灵力并尝试进阶称号
func (s *ChatService) GainExperience(userID uint) {
	var user model.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		return
	}

	user.Experience += 1

	// 称号算法
	newTitle := "初入龙屿的游侠"
	exp := user.Experience
	if exp > 500 {
		newTitle = "龙屿不灭的灵魂"
	} else if exp > 200 {
		newTitle = "洞悉因果的智者"
	} else if exp > 80 {
		newTitle = "守护龙火的圣骑"
	} else if exp > 30 {
		newTitle = "聆听真言的信徒"
	} else if exp > 10 {
		newTitle = "跋涉迷雾的旅者"
	}

	user.Title = newTitle
	global.DB.Save(&user)
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

// GenerateImageTask 提交任务
func (s *ChatService) GenerateImageTask(userID uint, ip string, prompt string, size string, resolution string, userToken string) (uint, uint, string, error) {
	todayStart := time.Now().Truncate(24 * time.Hour)

	var msgCount int64
	global.DB.Model(&model.Message{}).Where("user_id = ? AND created_at >= ?", userID, todayStart).Count(&msgCount)
	if msgCount == 0 {
		return 0, 0, "", errors.New("你今日尚未留下任何誓言，龙主无法捕捉你的神念")
	}

	var uCount int64
	global.DB.Model(&model.MagicRecord{}).Where("user_id = ? AND created_at >= ? AND prompt != ?", userID, todayStart, "Dragon Evolution Image").Count(&uCount)
	if uCount >= 2 {
		return 0, 0, "", errors.New("今日个人显像次数已达上限（2/2），请明日再试")
	}

	var gCount int64
	global.DB.Model(&model.MagicRecord{}).Where("created_at >= ?", todayStart).Count(&gCount)
	if gCount >= 20 {
		return 0, 0, "", errors.New("今日全站显像名额（20/20）已满，请明日请早")
	}

	record := &model.MagicRecord{
		UserID: userID,
		Prompt: prompt,
	}
	global.DB.Create(record)

	userMsg := &model.Message{
		Content: "✨ [龙语显像] " + prompt,
		UserID:  &userID,
	}
	global.DB.Create(userMsg)
	global.DB.Preload("User").First(userMsg, userMsg.ID)
	logic.GlobalHub.BroadcastMessage(userMsg)

	placeholderMsg := &model.Message{
		Content:   fmt.Sprintf("龙主正在凝聚灵力... [MAGIC_LOADING:%s]", prompt),
		IsAIReply: true,
	}
	global.DB.Create(placeholderMsg)
	global.DB.Preload("User").First(placeholderMsg, placeholderMsg.ID)
	logic.GlobalHub.BroadcastMessage(placeholderMsg)

	token := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(token)
	taskID, err := client.CreateImage(prompt, size, resolution)
	if err != nil {
		global.DB.Model(&model.Message{}).Where("id = ?", placeholderMsg.ID).Update("content", "龙主灵力不支，幻化中断... (提交失败)")
		global.DB.Unscoped().Delete(record)
		return 0, 0, "", err
	}

	// 启动异步任务 (传入 userToken 以便上传)
	go s.PollImageTaskResult(taskID, userID, placeholderMsg.ID, prompt, record.ID, userToken)

	// 更新每日显像任务
	ds := &DragonService{}
	ds.UpdateTaskProgress(userID, "generate", 1)

	return userMsg.ID, record.ID, taskID, nil
}

// PollImageTaskResult 轮询任务
func (s *ChatService) PollImageTaskResult(taskID string, userID uint, placeholderID uint, prompt string, recordID uint, userToken string) {
	token := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(token)

	for i := 0; i < 120; i++ {
		time.Sleep(5 * time.Second)
		res, err := client.GetTaskStatus(taskID)
		if err != nil { continue }

		if res.Data.Status == "completed" && len(res.Data.Result.Images) > 0 {
			remoteURL := res.Data.Result.Images[0].URL[0]
			cloudURL, _ := s.uploadToCloud(remoteURL, userToken)
			if cloudURL == "" { cloudURL = remoteURL }

			// 生成成功后，可以删掉占位消息，或者更新它
			global.DB.Unscoped().Delete(&model.Message{}, placeholderID)

			aiMsg := &model.Message{
				Content:          fmt.Sprintf("**咒语**: %s\n\n![](%s)", prompt, cloudURL),
				IsAIReply:        true,
				ReplyToMessageID: &placeholderID, 
			}
			global.DB.Create(aiMsg)
			global.DB.Preload("User").Preload("ReplyToMessage.User").First(aiMsg, aiMsg.ID)
			logic.GlobalHub.BroadcastMessage(aiMsg)
			return
		}
		if res.Data.Status == "failed" {
			global.DB.Model(&model.Message{}).Where("id = ?", placeholderID).Update("content", "龙主灵力涣散，此次幻化未能成形...")
			global.DB.Unscoped().Delete(&model.MagicRecord{}, recordID)
			return
		}
	}
	global.DB.Model(&model.Message{}).Where("id = ?", placeholderID).Update("content", "显像耗时过长，龙力已耗尽...")
	global.DB.Unscoped().Delete(&model.MagicRecord{}, recordID)
}

func (s *ChatService) uploadToCloud(remoteURL string, userToken string) (string, error) {
	resp, err := http.Get(remoteURL)
	if err != nil { return "", err }
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fmt.Sprintf("magic_%d.jpg", time.Now().UnixNano()))
	io.Copy(part, bytes.NewReader(data))
	writer.Close()

	req, _ := http.NewRequest("POST", "http://xiaolongya.cn:8888/dragon/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// 注入令牌
	if !strings.HasPrefix(userToken, "Bearer ") {
		userToken = "Bearer " + userToken
	}
	req.Header.Set("Authorization", userToken)

	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil { return "", err }
	defer resp2.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct { URL string `json:"url"` } `json:"data"`
	}
	json.NewDecoder(resp2.Body).Decode(&result)
	return result.Data.URL, nil
}
