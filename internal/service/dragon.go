package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/logic"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/apimart"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"math/rand"
	"strings"
	"time"
)

type DragonService struct{}

// StartLifeCycle 启动龙嗣生命周期管理 (动态消耗)
func (s *DragonService) StartLifeCycle() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			// 1. 处理饱食度消耗 (Common: 5, Rare: 8, Epic: 12)
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND hunger > 0", "common").
				Update("hunger", global.DB.Raw("CASE WHEN hunger >= 5 THEN hunger - 5 ELSE 0 END"))
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND hunger > 0", "rare").
				Update("hunger", global.DB.Raw("CASE WHEN hunger >= 8 THEN hunger - 8 ELSE 0 END"))
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND hunger > 0", "epic").
				Update("hunger", global.DB.Raw("CASE WHEN hunger >= 12 THEN hunger - 12 ELSE 0 END"))
			
			// 2. 处理心情值消耗 (Common: 3, Rare: 5, Epic: 8)
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND happiness > 0", "common").
				Update("happiness", global.DB.Raw("CASE WHEN happiness >= 3 THEN happiness - 3 ELSE 0 END"))
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND happiness > 0", "rare").
				Update("happiness", global.DB.Raw("CASE WHEN happiness >= 5 THEN happiness - 5 ELSE 0 END"))
			global.DB.Model(&model.Dragon{}).Where("rarity = ? AND happiness > 0", "epic").
				Update("happiness", global.DB.Raw("CASE WHEN happiness >= 8 THEN happiness - 8 ELSE 0 END"))
		}
	}()
}

// GetDragon 获取用户的龙
func (s *DragonService) GetDragon(userID uint) (*model.Dragon, error) {
	var dragon model.Dragon
	err := global.DB.Where("user_id = ? AND is_gone = ?", userID, false).First(&dragon).Error
	if err != nil {
		return nil, err
	}
	// 打开界面自动签到
	s.UpdateTaskProgress(userID, "sign_in", 1)
	return &dragon, nil
}

// GetItems 获取用户的物品
func (s *DragonService) GetItems(userID uint) ([]model.UserItem, error) {
	var items []model.UserItem
	err := global.DB.Where("user_id = ?", userID).Find(&items).Error
	return items, err
}

// UseItem 使用道具
func (s *DragonService) UseItem(userID uint, itemType string) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", errors.New("你还没有领养龙嗣")
	}

	var item model.UserItem
	if err := global.DB.Where("user_id = ? AND type = ?", userID, itemType).First(&item).Error; err != nil || item.Count <= 0 {
		return "", errors.New("你囊中并无此物")
	}

	msg := ""
	switch itemType {
	case "exp_pill":
		dragon.Exp += 50
		msg = "龙宝宝吞下了龙髓丹，周身灵力激荡，成长值显著提升！"
	case "sacrifice_stone":
		if dragon.Hunger < 30 {
			return "", errors.New("龙宝宝现在太虚弱了，无法承受献祭之石的力量")
		}
		dragon.Hunger -= 30
		dragon.Exp += 100
		msg = "献祭之石发出了幽暗的光芒，虽然饱食度骤降，但龙宝宝的力量得到了质的飞跃！"
	default:
		return "", errors.New("此物尚不知如何使用")
	}

	// 消耗道具
	item.Count--
	global.DB.Save(&item)
	global.DB.Save(dragon)

	return msg, nil
}

// TriggerRandomEvent 触发随机事件 (AI 故事 + 道具/属性奖励)
func (s *DragonService) TriggerRandomEvent(userID uint, actionType string) (string, error) {
	dragon, _ := s.GetDragon(userID)
	if dragon == nil { return "", nil }

	// 1. 确定奖励池
	rewardPool := []struct {
		Type  string // 'item', 'stat'
		Value string // 'exp_pill', 'sacrifice_stone', 'exp', 'happiness'
		Amt   int
		Name  string
	}{
		{Type: "stat", Value: "exp", Amt: 50, Name: "额外成长值"},
		{Type: "item", Value: "exp_pill", Amt: 1, Name: "龙髓丹"},
		{Type: "item", Value: "sacrifice_stone", Amt: 1, Name: "献祭之石"},
		{Type: "stat", Value: "happiness", Amt: 30, Name: "飞跃的好心情"},
	}

	reward := rewardPool[rand.Intn(len(rewardPool))]

	// 2. 召唤 AI 讲述奇遇
	apiToken := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(apiToken)
	
	systemPrompt := "你是一位龙屿世界的吟游诗人。请根据用户的行为（喂食或陪玩）和获得的奖励，写一段50字以内的极具诗意和画面感的奇遇描述。格式：[奇遇内容] 获得：[奖励名称]"
	userPrompt := fmt.Sprintf("动作：%s，龙的名字：%s，品质：%s，获得的奖励：%s", actionType, dragon.Name, dragon.Rarity, reward.Name)
	
	story, _ := client.GetChatCompletion("gpt-4o-mini", userPrompt, systemPrompt)
	if story == "" {
		story = fmt.Sprintf("龙宝宝在%s时显得格外兴奋，似乎触发了某种远古共鸣！获得：%s", actionType, reward.Name)
	}

	// 3. 发放奖励
	if reward.Type == "stat" {
		if reward.Value == "exp" { dragon.Exp += reward.Amt }
		if reward.Value == "happiness" { dragon.Happiness += reward.Amt }
		global.DB.Save(dragon)
	} else if reward.Type == "item" {
		var item model.UserItem
		if err := global.DB.Where("user_id = ? AND type = ?", userID, reward.Value).First(&item).Error; err != nil {
			item = model.UserItem{UserID: userID, Type: reward.Value, Count: reward.Amt}
			global.DB.Create(&item)
		} else {
			item.Count += reward.Amt
			global.DB.Save(&item)
		}
	}

	return story, nil
}

// Feed 喂食 (消耗 UserItem food)
func (s *DragonService) Feed(userID uint) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", errors.New("你还没有领养龙嗣")
	}

	if dragon.Hunger >= 100 {
		return "", errors.New("它已经吃不下了")
	}

	// 检查是否有龙粮
	var item model.UserItem
	if err := global.DB.Where("user_id = ? AND type = ?", userID, "food").First(&item).Error; err != nil || item.Count <= 0 {
		return "", errors.New("龙粮不足，请完成每日修行获取")
	}

	// 消耗龙粮
	item.Count--
	global.DB.Save(&item)

	dragon.Hunger += 20
	if dragon.Hunger > 100 {
		dragon.Hunger = 100
	}

	// 动态成长收益
	expGain := 10
	if dragon.Rarity == "rare" { expGain = 20 }
	if dragon.Rarity == "epic" { expGain = 40 }
	dragon.Exp += expGain

	now := time.Now()
	dragon.LastFedAt = &now

	if err := global.DB.Save(dragon).Error; err != nil {
		return "", err
	}

	// 更新任务进度
	s.UpdateTaskProgress(userID, "feed", 1)

	// 30% 概率触发随机事件
	eventMsg := "无事发生"
	if rand.Float32() < 0.3 {
		eventMsg, _ = s.TriggerRandomEvent(userID, "feed")
	}

	return eventMsg, nil
}

// Play 陪玩 (固定时间间隔 10 分钟)
func (s *DragonService) Play(userID uint) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", errors.New("你还没有领养龙嗣")
	}

	if dragon.Happiness >= 100 {
		return "", errors.New("它现在心情好极了")
	}

	// 检查时间间隔
	if dragon.LastPlayedAt != nil {
		if time.Since(*dragon.LastPlayedAt) < 10*time.Minute {
			wait := 10 - int(time.Since(*dragon.LastPlayedAt).Minutes())
			return "", fmt.Errorf("龙宝宝玩累了，请 %d 分钟后再来", wait)
		}
	}

	dragon.Happiness += 15
	if dragon.Happiness > 100 {
		dragon.Happiness = 100
	}

	// 动态成长收益
	expGain := 10
	if dragon.Rarity == "rare" { expGain = 20 }
	if dragon.Rarity == "epic" { expGain = 40 }
	dragon.Exp += expGain

	now := time.Now()
	dragon.LastPlayedAt = &now

	if err := global.DB.Save(dragon).Error; err != nil {
		return "", err
	}

	// 更新任务进度
	s.UpdateTaskProgress(userID, "play", 1)

	// 30% 概率触发随机事件
	eventMsg := "无事发生"
	if rand.Float32() < 0.3 {
		eventMsg, _ = s.TriggerRandomEvent(userID, "play")
	}

	return eventMsg, nil
}

// Rename 改名
func (s *DragonService) Rename(userID uint, name string) error {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return err
	}
	dragon.Name = name
	return global.DB.Save(dragon).Error
}

// ShareToChat 将龙的照片分享到聊天广场 (每日限1次)
func (s *DragonService) ShareToChat(userID uint) error {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return errors.New("你还没有领养龙嗣")
	}

	if dragon.ImageURL == "" || dragon.ImageURL == "[GENERATING]" {
		return errors.New("你的龙宝宝还没有幻化真身，快去显像吧")
	}

	// 校验每日分享次数
	today := time.Now().Format("2006-01-02")
	var task model.UserTask
	if err := global.DB.Where("user_id = ? AND task_type = ? AND date = ?", userID, "share", today).First(&task).Error; err == nil {
		if task.Progress >= 1 {
			return errors.New("今日已在广场展示过英姿，请明日再来")
		}
	}

	// 构造展示消息
	content := fmt.Sprintf("✨ [真身显现] 游侠展示了他的龙宝宝 **%s**！\n\n![](%s)", dragon.Name, dragon.ImageURL)
	
	msg := &model.Message{
		Content: content,
		UserID:  &userID,
	}
	msg.CreatedAt = time.Now() // 显式设置时间，确保排序正确且广播有时间

	if err := global.DB.Create(msg).Error; err != nil {
		return err
	}

	// 广播前确保关联数据完整
	global.DB.Preload("User").First(msg, msg.ID)
	logic.GlobalHub.BroadcastMessage(msg)

	// 更新每日分享任务
	s.UpdateTaskProgress(userID, "share", 1)
	return nil
}

// UpdateTaskProgress 更新任务进度
func (s *DragonService) UpdateTaskProgress(userID uint, taskType string, amount int) {
	today := time.Now().Format("2006-01-02")
	var task model.UserTask
	
	if err := global.DB.Where("user_id = ? AND task_type = ? AND date = ?", userID, taskType, today).First(&task).Error; err != nil {
		// 创建新任务记录
		task = model.UserTask{
			UserID:      userID,
			TaskType:    taskType,
			Progress:    amount,
			MaxProgress: 1, 
			Date:        today,
		}
		global.DB.Create(&task)
	} else {
		if task.Progress < task.MaxProgress {
			task.Progress += amount
			if task.Progress > task.MaxProgress {
				task.Progress = task.MaxProgress
			}
			global.DB.Save(&task)
		}
	}
}

// GetDailyTasks 获取今日任务 (根据稀有度动态生成)
func (s *DragonService) GetDailyTasks(userID uint) []model.UserTask {
	dragon, _ := s.GetDragon(userID)
	rarity := "common"
	if dragon != nil {
		rarity = dragon.Rarity
	}

	today := time.Now().Format("2006-01-02")
	
	// 任务配置: {基础次数, 基础龙粮奖励, 基础经验奖励}
	configs := map[string][]int{
		"sign_in":  {1, 5, 10},   
		"chat":     {5, 5, 15},   
		"generate": {1, 10, 30},   
		"share":    {1, 10, 20},   
		"feed":     {3, 5, 10},
		"play":     {2, 5, 15},
		"fortune":  {1, 5, 10},
	}

	// 难度倍率
	diffMult := 1.0 
	if rarity == "rare" { 
		diffMult = 2.0
	}
	if rarity == "epic" { 
		diffMult = 3.5
	}

	var tasks []model.UserTask
	for tt, cfg := range configs {
		var task model.UserTask
		maxProgress := int(float64(cfg[0]) * diffMult)
		if maxProgress < 1 { maxProgress = 1 }

		if err := global.DB.Where("user_id = ? AND task_type = ? AND date = ?", userID, tt, today).First(&task).Error; err != nil {
			task = model.UserTask{
				UserID:      userID,
				TaskType:    tt,
				Progress:    0,
				MaxProgress: maxProgress,
				Date:        today,
			}
			global.DB.Create(&task)
		} else {
			task.MaxProgress = maxProgress
			task.TaskType = tt 
			global.DB.Save(&task)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

// ClaimTaskReward 领取任务奖励
func (s *DragonService) ClaimTaskReward(userID uint, taskID uint) (string, error) {
	var task model.UserTask
	if err := global.DB.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		return "", errors.New("修行任务不存在")
	}

	if task.Progress < task.MaxProgress {
		return "", errors.New("修行尚未圆满，继续努力吧")
	}

	if task.IsClaimed {
		return "", errors.New("此项天恩奖励已领受")
	}

	dragon, _ := s.GetDragon(userID)
	rarity := "common"
	if dragon != nil { rarity = dragon.Rarity }

	rewardMult := 1.0
	if rarity == "rare" { rewardMult = 2.5 }
	if rarity == "epic" { rewardMult = 5.0 }

	rewards := map[string][]int{
		"sign_in":  {5, 10},  
		"chat":     {5, 15},
		"generate": {10, 30},
		"share":    {10, 20},
		"feed":     {5, 10},
		"play":     {5, 15},
		"fortune":  {5, 10},
	}

	cfg := rewards[task.TaskType]
	foodCount := int(float64(cfg[0]) * rewardMult)
	expGained := int(float64(cfg[1]) * rewardMult)

	// 发放奖励
	if foodCount > 0 {
		var item model.UserItem
		if err := global.DB.Where("user_id = ? AND type = ?", userID, "food").First(&item).Error; err != nil {
			item = model.UserItem{UserID: userID, Type: "food", Count: foodCount}
			global.DB.Create(&item)
		} else {
			item.Count += foodCount
			global.DB.Save(&item)
		}
	}

	if expGained > 0 && dragon != nil {
		dragon.Exp += expGained
		global.DB.Save(dragon)
	}

	task.IsClaimed = true
	global.DB.Save(&task)

	return fmt.Sprintf("领取成功！获得龙粮x%d, 经验x%d", foodCount, expGained), nil
}

// GenerateImage 提交显像任务
func (s *DragonService) GenerateImage(userID uint, userToken string) error {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return err
	}

	todayStart := time.Now().Truncate(24 * time.Hour)
	var gCount int64
	global.DB.Model(&model.MagicRecord{}).Where("created_at >= ?", todayStart).Count(&gCount)
	if gCount >= 20 {
		return errors.New("今日龙主显像灵力已耗尽（20/20），请明日再试")
	}

	record := &model.MagicRecord{
		UserID: userID,
		Prompt: "Dragon Evolution Image",
	}
	global.DB.Create(record)

	stages := []string{"mystical egg", "cute baby dragon", "majestic young dragon", "powerful ancient dragon"}
	currentStage := stages[0]
	if dragon.Stage < len(stages) {
		currentStage = stages[dragon.Stage]
	}
	
	prompt := fmt.Sprintf("%s, %s, %s, high detail, fantasy style, glowing elements, 8k resolution", dragon.Rarity, currentStage, dragon.BasePrompt)

	global.DB.Model(&model.Dragon{}).Where("id = ?", dragon.ID).Update("image_url", "[GENERATING]")

	apiToken := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(apiToken)
	taskID, err := client.CreateImage(prompt, "1:1", "1k")
	if err != nil {
		return fmt.Errorf("显像任务提交失败: %v", err)
	}

	go s.PollDragonImage(taskID, dragon.ID, record.ID, userToken)
	s.UpdateTaskProgress(userID, "generate", 1)

	return nil
}

func (s *DragonService) PollDragonImage(taskID string, dragonID uint, recordID uint, userToken string) {
	apiToken := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(apiToken)

	for i := 0; i < 60; i++ {
		time.Sleep(10 * time.Second)
		res, err := client.GetTaskStatus(taskID)
		if err != nil {
			continue
		}

		if res.Data.Status == "completed" && len(res.Data.Result.Images) > 0 {
			remoteURL := res.Data.Result.Images[0].URL[0]
			cloudURL, err := s.uploadToCloud(remoteURL, userToken)
			if err != nil {
				cloudURL = remoteURL
			}

			global.DB.Model(&model.Dragon{}).Where("id = ?", dragonID).Update("image_url", cloudURL)
			return
		}
		if res.Data.Status == "failed" {
			global.DB.Model(&model.Dragon{}).Where("id = ?", dragonID).Update("image_url", "") 
			global.DB.Unscoped().Delete(&model.MagicRecord{}, recordID) 
			return
		}
	}
	global.DB.Model(&model.Dragon{}).Where("id = ?", dragonID).Update("image_url", "") 
	global.DB.Unscoped().Delete(&model.MagicRecord{}, recordID)
}

func (s *DragonService) uploadToCloud(remoteURL string, userToken string) (string, error) {
	resp, err := http.Get(remoteURL)
	if err != nil { return "", err }
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", fmt.Sprintf("dragon_%d.jpg", time.Now().UnixNano()))
	io.Copy(part, bytes.NewReader(data))
	writer.Close()

	req, _ := http.NewRequest("POST", "http://xiaolongya.cn:8888/dragon/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
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
