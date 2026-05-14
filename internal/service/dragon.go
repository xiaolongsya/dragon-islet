package service

import (
	"bytes"
	"dragon-islet/internal/global"
	"dragon-islet/internal/logic"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/apimart"
	"dragon-islet/pkg/deepseek"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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

			// 3. 处理特殊状态效果
			var activeStatuses []model.DragonStatus
			global.DB.Where("expires_at IS NULL OR expires_at > ?", time.Now()).Find(&activeStatuses)
			for _, st := range activeStatuses {
				// 解析效果，如 "happiness-10"
				if strings.Contains(st.Effect, "-") {
					parts := strings.Split(st.Effect, "-")
					if len(parts) == 2 {
						attr := parts[0]
						val, _ := strconv.Atoi(parts[1])
						global.DB.Model(&model.Dragon{}).Where("id = ?", st.DragonID).
							Update(attr, global.DB.Raw(fmt.Sprintf("CASE WHEN %s >= %d THEN %s - %d ELSE 0 END", attr, val, attr, val)))
					}
				} else if strings.Contains(st.Effect, "+") {
					parts := strings.Split(st.Effect, "+")
					if len(parts) == 2 {
						attr := parts[0]
						val, _ := strconv.Atoi(parts[1])
						// 处理最大值限制 (MaxHappiness, MaxHunger)
						maxAttr := "100"
						if attr == "happiness" || attr == "hunger" {
							maxAttr = "max_" + attr
						}
						global.DB.Model(&model.Dragon{}).Where("id = ?", st.DragonID).
							Update(attr, global.DB.Raw(fmt.Sprintf("CASE WHEN %s + %d <= %s THEN %s + %d ELSE %s END", attr, val, maxAttr, attr, val, maxAttr)))
					}
				}
			}

			// 4. 清理已过期的状态
			global.DB.Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).Unscoped().Delete(&model.DragonStatus{})
		}
	}()
}

// GetDragon 获取用户的龙
func (s *DragonService) GetDragon(userID uint) (*model.Dragon, error) {
	var dragon model.Dragon
	err := global.DB.Preload("Statuses", "expires_at IS NULL OR expires_at > ?", time.Now()).
		Where("user_id = ? AND is_gone = ?", userID, false).First(&dragon).Error
	if err != nil {
		return nil, err
	}
	// 打开界面自动签到
	s.UpdateTaskProgress(userID, "sign_in", 1)

	// 处理每日奇遇次数重置 (凌晨3点重置)
	now := time.Now()
	adjNow := now.Add(-3 * time.Hour)
	var adjLast time.Time
	if dragon.LastEventAt != nil {
		adjLast = dragon.LastEventAt.Add(-3 * time.Hour)
	}

	if dragon.LastEventAt == nil || adjLast.Format("2006-01-02") != adjNow.Format("2006-01-02") {
		dragon.DailyEventCount = 0
		global.DB.Model(&dragon).Update("daily_event_count", 0)
	}

	return &dragon, nil
}

// GetItems 获取用户的物品
func (s *DragonService) GetItems(userID uint) ([]model.UserItem, error) {
	var items []model.UserItem
	err := global.DB.Where("user_id = ?", userID).Find(&items).Error
	return items, err
}

// UseItem 使用道具
func (s *DragonService) UseItem(userID uint, itemType string, itemName string) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", errors.New("你还没有领养龙嗣")
	}

	var item model.UserItem
	query := global.DB.Where("user_id = ?", userID)
	if itemName != "" {
		query = query.Where("name = ?", itemName)
	} else {
		query = query.Where("type = ?", itemType)
	}

	if err := query.First(&item).Error; err != nil || item.Count <= 0 {
		return "", errors.New("你囊中并无此物")
	}

	msg := ""
	if itemType == "custom" || (itemType == "" && itemName != "") {
		if item.Category != "usable" {
			return "", errors.New("此物仅供收藏，无法直接使用")
		}
		// 默认自定义道具效果：提升少量经验与心情
		dragon.Exp += 15
		dragon.Happiness += 10
		if dragon.Happiness > 100 {
			dragon.Happiness = 100
		}
		msg = fmt.Sprintf("你使用了【%s】，龙宝宝感受到了一丝奇特的灵力，成长值与心情都有所提升。", item.Name)
	} else {
		switch itemType {
		case "food":
			// 虽然前端有单独的投喂，但也可以通过包裹使用
			dragon.Hunger += 20
			if dragon.Hunger > 100 {
				dragon.Hunger = 100
			}
			msg = "龙宝宝吃下了龙粮，饱食度得到了补充。"
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
	}

	// 消耗道具
	item.Count--
	global.DB.Save(&item)
	global.DB.Save(dragon)

	return msg, nil
}

// ReleaseDragon 放生龙 (需要密码校验)
func (s *DragonService) ReleaseDragon(userID uint, password string) error {
	// 1. 获取用户
	var user model.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		return errors.New("游侠身份验证失败")
	}

	// 2. 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return errors.New("密语校验失败，无法解除契约")
	}

	// 3. 获取龙
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return errors.New("你并无契约龙嗣可以放生")
	}

	// 4. 更新状态
	dragon.IsGone = true
	if err := global.DB.Save(dragon).Error; err != nil {
		return err
	}

	// 5. 自动在广场播报
	releaseMsg := fmt.Sprintf("✨ 游侠 **%s** 解除了与龙嗣 **%s** 的契约。只见那龙化作一道霞光，隐入云端深处，回归了自然的怀抱...", user.Username, dragon.Name)
	msg := &model.Message{
		Content:   releaseMsg,
		IsAIReply: true, // 标记为 AI/系统 消息样式
	}
	global.DB.Create(msg)
	logic.GlobalHub.BroadcastMessage(msg)

	return nil
}

// TriggerRandomEvent 触发随机事件 (AI 故事 + 道具/属性奖励)
func (s *DragonService) TriggerRandomEvent(userID uint, actionType string) (string, error) {
	dragon, _ := s.GetDragon(userID)
	if dragon == nil {
		return "", nil
	}

	// 检查是否跨越凌晨3点，执行重置
	now := time.Now()
	adjNow := now.Add(-3 * time.Hour)
	var adjLast time.Time
	if dragon.LastEventAt != nil {
		adjLast = dragon.LastEventAt.Add(-3 * time.Hour)
	}
	if dragon.LastEventAt == nil || adjLast.Format("2006-01-02") != adjNow.Format("2006-01-02") {
		dragon.DailyEventCount = 0
	}

	// 每日限额 10 次
	if dragon.DailyEventCount >= 10 {
		return "今日已阅尽世间奇遇，且去修行，待明日旭日东升，再寻机缘。", nil
	}

	// 0. 计算状态加成 (atk 增加胜率, luck 增加好运)
	atkBonus := 0
	luckBonus := 0
	statusInfo := ""
	for _, st := range dragon.Statuses {
		statusInfo += fmt.Sprintf("[%s: %s], ", st.Name, st.Desc)
		if strings.Contains(st.Effect, "atk+") {
			val, _ := strconv.Atoi(strings.Split(st.Effect, "atk+")[1])
			atkBonus += val
		} else if strings.Contains(st.Effect, "luck+") {
			val, _ := strconv.Atoi(strings.Split(st.Effect, "luck+")[1])
			luckBonus += val
		}
	}
	if statusInfo == "" {
		statusInfo = "无特殊状态"
	}

	// 1. 确定动态奖励池与权重
	isDistressed := dragon.Hunger < 20 || dragon.Happiness < 20

	type Reward struct {
		Type   string
		Value  string
		Amt    int
		Name   string
		Weight int
	}

	rewardPool := []Reward{
		{Type: "stat", Value: "exp", Amt: 50, Name: "额外成长值", Weight: 30},
		{Type: "item", Value: "exp_pill", Amt: 1, Name: "龙髓丹", Weight: 10},
		{Type: "stat", Value: "happiness", Amt: 30, Name: "愉悦的心情", Weight: 30},
		{Type: "status", Value: "good_status", Amt: 0, Name: "增益状态", Weight: 30},
	}

	if isDistressed {
		// 状态不佳时，加入负面事件
		badWeight := 40 - atkBonus - luckBonus // atk 和 luck 降低厄运概率
		if badWeight < 5 {
			badWeight = 5
		}
		rewardPool = append(rewardPool,
			Reward{Type: "stat", Value: "exp", Amt: -30, Name: "灵力溃散", Weight: badWeight},
			Reward{Type: "status", Value: "bad_status", Amt: 0, Name: "负面诅咒", Weight: badWeight + 10},
		)
	} else {
		// 状态良好时，额外好运加成
		for i := range rewardPool {
			rewardPool[i].Weight += luckBonus / 2
		}
	}

	// 加权随机抽取
	totalWeight := 0
	for _, r := range rewardPool {
		totalWeight += r.Weight
	}
	randWeight := rand.Intn(totalWeight)
	var reward Reward
	for _, r := range rewardPool {
		randWeight -= r.Weight
		if randWeight < 0 {
			reward = r
			break
		}
	}

	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))

	stateDesc := "状态良好"
	if isDistressed {
		stateDesc = "极度饥饿或心情低落"
	}

	systemPrompt := fmt.Sprintf(`你是一位龙屿世界的吟游诗人。
当前龙的状态：%s
当前活跃加成：%s (攻击加成: %d, 幸运加成: %d)
请根据用户的行为和获得的奖励（可能是惩罚），写一段 50 字以内的奇遇描述。
如果奖励是好的且攻击/幸运很高，可以描述为龙凭借自身能力克服了困难。
严禁使用 Emoji。`, stateDesc, statusInfo, atkBonus, luckBonus)
	systemPrompt += `格式要求：
1. 奇遇描述请保持在 50 字以内。
2. 在描述之后，换行输出奖励指令：
获得物品：[物品名] (说明：[详细描述])
得到[数值]点成长值

注：
- 物品说明中如果包含“使用后可...”则该物品将被归类为“可以使用”，否则为“收藏品”。
- 如果没有获得物品，可以不输出“获得物品”行。`

	userPrompt := fmt.Sprintf("动作：%s，龙的名字：%s，品质：%s，获得的奖励：%s", actionType, dragon.Name, dragon.Rarity, reward.Name)

	story, _ := dsClient.Chat([]deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if story == "" {
		story = fmt.Sprintf("龙宝宝在%s时显得格外兴奋，似乎触发了某种远古共鸣！获得：%s", actionType, reward.Name)
	}

	// 3. 解析奖励指令
	// a. 成长值
	if strings.Contains(story, "得到") && strings.Contains(story, "点成长值") {
		parts := strings.Split(story, "得到")
		if len(parts) > 1 {
			valStr := strings.Split(parts[1], "点成长值")[0]
			valStr = strings.TrimSpace(valStr)
			val, _ := strconv.Atoi(valStr)
			dragon.Exp += val
		}
	}

	// b. 物品
	if strings.Contains(story, "获得物品：") {
		line := ""
		for _, l := range strings.Split(story, "\n") {
			if strings.Contains(l, "获得物品：") {
				line = l
				break
			}
		}
		if line != "" {
			parts := strings.Split(line, "获得物品：")
			if len(parts) > 1 {
				content := strings.TrimSpace(parts[1])
				name := strings.Split(content, "(")[0]
				name = strings.TrimSpace(name)
				desc := ""
				if strings.Contains(content, "说明：") {
					desc = strings.Split(content, "说明：")[1]
					desc = strings.TrimSuffix(desc, ")")
					desc = strings.TrimSpace(desc)
				}

				category := "collectible"
				if strings.Contains(desc, "使用后") || strings.Contains(desc, "增加") {
					category = "usable"
				}

				var item model.UserItem
				// 具有相同名称的物品堆叠
				if err := global.DB.Where("user_id = ? AND name = ?", userID, name).First(&item).Error; err != nil {
					item = model.UserItem{
						UserID:   userID,
						Type:     "custom",
						Name:     name,
						Desc:     desc,
						Category: category,
						Count:    1,
					}
					global.DB.Create(&item)
				} else {
					item.Count++
					global.DB.Save(&item)
				}
				fmt.Printf("[Random Event] User %d obtained item: %s (%s)\n", userID, name, category)
			}
		}
	}

	// c. 状态 (保留原有的状态解析)
	s.parseAndApplyStatus(dragon.ID, story)

	// 处理预定义的奖励 (如果 AI 没有给出特定指令，则兜底)
	if reward.Type == "stat" && !strings.Contains(story, "成长值") {
		if reward.Value == "exp" {
			dragon.Exp += reward.Amt
		}
		if reward.Value == "happiness" {
			dragon.Happiness += reward.Amt
		}
	} else if reward.Type == "item" && !strings.Contains(story, "获得物品") {
		var item model.UserItem
		if err := global.DB.Where("user_id = ? AND type = ?", userID, reward.Value).First(&item).Error; err != nil {
			item = model.UserItem{UserID: userID, Type: reward.Value, Name: reward.Name, Category: "usable", Count: reward.Amt}
			global.DB.Create(&item)
		} else {
			item.Count += reward.Amt
			global.DB.Save(&item)
		}
	}
	global.DB.Save(dragon)

	// 增加计数
	now = time.Now()
	dragon.DailyEventCount++
	dragon.LastEventAt = &now
	global.DB.Save(dragon)

	return story, nil
}

// Feed 喂食 (消耗 UserItem food)
// Evolve 进化
func (s *DragonService) Evolve(userID uint, userToken string) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", err
	}
	if dragon.Stage >= 4 {
		return "", fmt.Errorf("已是真龙之躯，无可再进")
	}

	// 对应前端 [100, 300, 800, 2000]
	expMap := []int{100, 300, 800, 2000}
	nextExp := expMap[dragon.Stage]
	if dragon.Exp < nextExp {
		return "", fmt.Errorf("成长值不足，尚需修行 (当前: %d/%d)", dragon.Exp, nextExp)
	}

	stages := []string{"龙蛋", "幼龙", "青年龙", "壮年龙", "真龙"}
	oldStageName := stages[dragon.Stage]

	// 晋升阶段
	dragon.Stage++
	dragon.Exp = 0
	dragon.MaxHunger += 50
	dragon.MaxHappiness += 50
	dragon.Hunger = dragon.MaxHunger
	dragon.Happiness = dragon.MaxHappiness

	newStageName := stages[dragon.Stage]

	// 自动进化 BasePrompt
	evolvedPrompt := s.triggerPromptEvolution(dragon.BasePrompt, oldStageName, newStageName)
	if evolvedPrompt != "" {
		dragon.BasePrompt = evolvedPrompt
	}

	if err := global.DB.Save(dragon).Error; err != nil {
		return "", err
	}
	fmt.Printf("[DB Update] Dragon %d (User %d) evolved to stage %d. New BasePrompt: %s\n", dragon.ID, userID, dragon.Stage, dragon.BasePrompt)

	// 自动触发新阶真身显像
	go s.GenerateImage(userID, userToken, "")

	return fmt.Sprintf("突破成功！你的龙嗣已进化为【%s】，其基因图谱已自适应演变。正在为你重塑新阶真身...", newStageName), nil
}

// triggerPromptEvolution 调用 AI 进化提示词
func (s *DragonService) triggerPromptEvolution(oldPrompt, oldStage, newStage string) string {
	apiToken := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(apiToken)

	systemPrompt := `You are a dragon genetic specialist. Rewrite the dragon's appearance description to reflect its evolution.
STAGES & TRANSITIONS:
- Hatchling (幼龙): Tiny, cute, newborn dragon JUST broken out of its eggshell, pieces of shell around it, curious large eyes, slightly wet and clumsy.
- Young Dragon (青年龙): A small, sleek adolescent dragon. Agile, slender, approximately the size of a horse. Sharp growing horns and wings.
- Adult Dragon (壮年龙): A massive, imposing large dragon. Colossal body, powerful thick scales, wide wingspan, towering over the landscape or buildings.
- True Dragon (真龙): The ultimate divine form. Incredibly handsome, majestic, radiating golden light or cosmic energy. It is of mountain-like or cosmic scale, god-like presence.

Requirements:
1. STRICTLY PRESERVE original colors and elemental features: If the current description mentions "blue scales", "icy breath", or "golden eyes", these MUST be maintained in the new stage.
2. EMPHASIZE EXTREME SCALE DIFFERENCE: 
   - Hatchlings: Palm-sized, with eggshells.
   - Young: Small/Medium size (horse-sized).
   - Adult: Giant/Large size (building-sized).
   - True: God-like/Cosmic size (mountain-sized).
3. STYLE: Ensure the final stage (True Dragon) is exceptionally handsome, majestic, and powerful.
4. Output ONLY the new description in English, under 100 words. No introductory text.
`
	userPrompt := fmt.Sprintf("Current Stage: %s, Next Stage: %s. Current Description: %s", oldStage, newStage, oldPrompt)

	newPrompt, err := client.GetChatCompletion("gpt-4o-mini", userPrompt, systemPrompt)
	if err != nil || newPrompt == "" {
		fmt.Printf("[Evolution Error] AI failed to respond: %v\n", err)
		// Fallback: ensure no "egg" is carried over
		return fmt.Sprintf("A powerful and majestic %s, radiating elemental energy with shimmering scales and a regal presence.", newStage)
	}
	fmt.Printf("[Evolution] Old: %s -> New: %s\n", oldPrompt, newPrompt)
	return newPrompt
}

func (s *DragonService) Feed(userID uint) (string, error) {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return "", errors.New("你还没有领养龙嗣")
	}

	if dragon.Hunger >= dragon.MaxHunger {
		return "", fmt.Errorf("它已经吃不下了 (上限: %d)", dragon.MaxHunger)
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
	if dragon.Rarity == "rare" {
		expGain = 20
	}
	if dragon.Rarity == "epic" {
		expGain = 40
	}
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

	if dragon.Happiness >= dragon.MaxHappiness {
		return "", fmt.Errorf("它已经玩得很开心了 (上限: %d)", dragon.MaxHappiness)
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
	if dragon.Rarity == "rare" {
		expGain = 20
	}
	if dragon.Rarity == "epic" {
		expGain = 40
	}
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

	// 已去除每日分享次数校验，允许游侠多次展示英姿

	// 构造展示消息
	content := fmt.Sprintf("[真身显现] 游侠展示了他的龙宝宝 **%s**！\n\n![](%s)", dragon.Name, dragon.ImageURL)

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

// UpdateTaskProgress 更新任务进度 (凌晨3点刷新逻辑)
// UpdateTaskProgress 更新任务进度 (凌晨3点刷新逻辑)
func (s *DragonService) UpdateTaskProgress(userID uint, taskType string, amount int) {
	today := time.Now().Add(-3 * time.Hour).Format("2006-01-02")
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

	today := time.Now().Add(-3 * time.Hour).Format("2006-01-02")

	// 任务配置: {基础次数, 基础龙粮奖励, 基础经验奖励}
	type taskCfg struct {
		key  string
		vals []int
	}
	orderedConfigs := []taskCfg{
		{"sign_in", []int{1, 5, 10}},
		{"chat", []int{5, 5, 15}},
		{"generate", []int{1, 10, 30}},
		{"share", []int{1, 10, 20}},
		{"feed", []int{3, 5, 10}},
		{"play", []int{2, 5, 15}},
		{"fortune", []int{1, 5, 10}},
	}

	// 难度倍率
	diffMult := 1.0
	if rarity == "rare" {
		diffMult = 2.0
	}
	if rarity == "epic" {
		diffMult = 3.5
	}

	var existingTasks []model.UserTask
	global.DB.Where("user_id = ? AND date = ?", userID, today).Find(&existingTasks)

	taskMap := make(map[string]*model.UserTask)
	for i := range existingTasks {
		taskMap[existingTasks[i].TaskType] = &existingTasks[i]
	}

	var tasks []model.UserTask
	for _, cfg := range orderedConfigs {
		tt := cfg.key
		vals := cfg.vals
		maxProgress := int(float64(vals[0]) * diffMult)
		if maxProgress < 1 {
			maxProgress = 1
		}

		if task, ok := taskMap[tt]; ok {
			// 如果任务已存在，检查是否需要更新 MaxProgress
			if task.MaxProgress != maxProgress {
				task.MaxProgress = maxProgress
				global.DB.Select("MaxProgress").Save(task)
			}
			tasks = append(tasks, *task)
		} else {
			// 如果任务不存在，创建新任务
			newTask := model.UserTask{
				UserID:      userID,
				TaskType:    tt,
				Progress:    0,
				MaxProgress: maxProgress,
				Date:        today,
			}
			if err := global.DB.Create(&newTask).Error; err == nil {
				tasks = append(tasks, newTask)
			}
		}
	}
	return tasks
}

// CheckAllTasksCompleted 检查今日任务是否全部完成 (用于 4K 显像限制)
func (s *DragonService) CheckAllTasksCompleted(userID uint) bool {
	tasks := s.GetDailyTasks(userID)
	if len(tasks) == 0 {
		return false
	}
	for _, t := range tasks {
		if t.Progress < t.MaxProgress {
			return false
		}
	}
	return true
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
	if dragon != nil {
		rarity = dragon.Rarity
	}

	rewardMult := 1.0
	if rarity == "rare" {
		rewardMult = 2.5
	}
	if rarity == "epic" {
		rewardMult = 5.0
	}

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
func (s *DragonService) GenerateImage(userID uint, userToken string, refImageURL string) error {
	dragon, err := s.GetDragon(userID)
	if err != nil {
		return err
	}

	// 每日显像限制也跟随 3点刷新逻辑
	todayStart := time.Now().Add(-3 * time.Hour).Truncate(24 * time.Hour).Add(3 * time.Hour)
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

	stages := []string{"mystical dragon egg", "newborn hatchling dragon with broken eggshells", "sleek small adolescent dragon", "colossal large adult dragon", "divine majestic handsome true dragon"}
	sizeKeywords := []string{
		"small size, sitting on a pedestal",
		"tiny palm-sized, emerging from a shell, macro photography",
		"small-medium size, horse-sized, agile build, forest background",
		"massive colossal scale, building-sized, powerful presence, mountains in background",
		"god-like cosmic scale, mountain-sized, handsome facial features, radiant energy, cinematic masterpiece",
	}

	currentStage := stages[0]
	currentSize := sizeKeywords[0]
	if dragon.Stage >= 0 && dragon.Stage < len(stages) {
		currentStage = stages[dragon.Stage]
		currentSize = sizeKeywords[dragon.Stage]
	}

	prompt := fmt.Sprintf("%s, %s, %s stage dragon, %s, extreme scale, comparison to environment, cinematic lighting, high detail, fantasy masterpiece, 8k", dragon.Rarity, currentSize, currentStage, dragon.BasePrompt)

	// 设置为生成中，触发前端动画
	global.DB.Model(&model.Dragon{}).Where("id = ?", dragon.ID).Update("image_url", "[GENERATING]")

	var refImages []string
	// 只有在手动触发且不是进化的初始显像时（即 prompt 为空或非进化阶段）才考虑参考图？
	// 这里根据用户需求：进化时不弄参考图了。
	// 如果是手动显像（GenerateImage 被 handler 调用且 refImageURL 为空），则不带参考图。
	if refImageURL != "" && !strings.HasPrefix(refImageURL, "[") {
		refImages = []string{refImageURL}
	}

	apiToken := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(apiToken)
	taskID, err := client.CreateImage(prompt, "1:1", "1k", refImages)
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
	if err != nil {
		return "", err
	}
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
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	json.NewDecoder(resp2.Body).Decode(&result)
	return result.Data.URL, nil
}

// GetDragonChatHistory 获取龙的私聊记录
func (s *DragonService) GetDragonChatHistory(userID uint, page, pageSize int) ([]model.DragonMessage, error) {
	var dragon model.Dragon
	if err := global.DB.Where("user_id = ? AND is_gone = false", userID).First(&dragon).Error; err != nil {
		return nil, err
	}
	var msgs []model.DragonMessage
	offset := (page - 1) * pageSize
	err := global.DB.Where("dragon_id = ?", dragon.ID).Order("created_at desc").Offset(offset).Limit(pageSize).Find(&msgs).Error

	// 反转回正序显示
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, err
}

// ChatWithDragon 与龙进行私聊
func (s *DragonService) PrepareDragonChatStream(userID uint, content string) (*http.Response, *model.Dragon, error) {
	var dragon model.Dragon
	if err := global.DB.Where("user_id = ? AND is_gone = false", userID).First(&dragon).Error; err != nil {
		return nil, nil, errors.New("你还没有契约龙嗣")
	}

	// 1. 记录用户消息
	userMsg := model.DragonMessage{
		DragonID: dragon.ID,
		UserID:   userID,
		Role:     "user",
		Content:  content,
	}
	global.DB.Create(&userMsg)

	// 2. 获取最近上下文
	var history []model.DragonMessage
	global.DB.Where("dragon_id = ?", dragon.ID).Order("created_at desc").Limit(20).Find(&history)

	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	// 3. 构造 AI 指令
	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))

	// 加载当前状态
	var currentStatuses []model.DragonStatus
	global.DB.Where("dragon_id = ? AND (expires_at IS NULL OR expires_at > ?)", dragon.ID, time.Now()).Find(&currentStatuses)
	statusStr := "None"
	if len(currentStatuses) > 0 {
		statusStr = ""
		for _, s := range currentStatuses {
			statusStr += fmt.Sprintf("[%s: %s], ", s.Name, s.Desc)
		}
	}

	systemPrompt := fmt.Sprintf(`You are a dragon named "%s". 
Rarity: %s, Stage: %d.
Stats: Hunger %d/%d, Happiness %d/%d, Exp %d.
Current Statuses: %s.
Memory: %s.
Soul: %s.

Guidelines:
1. Concise, mystical tone, no emojis.
2. If the interaction justifies a NEW status (e.g. injured, extremely happy, found a friend), append a directive at the END of your message:
获得状态：[Name] | [DurationHours] | [Effect: attribute+/-value] | [Description]
Example: 获得状态：小确幸 | 2 | happiness+5 | 感受到了主人的关怀，心里暖暖的。
Supported attributes: hunger, happiness, exp.`,
		dragon.Name, dragon.Rarity, dragon.Stage,
		dragon.Hunger, dragon.MaxHunger, dragon.Happiness, dragon.MaxHappiness, dragon.Exp,
		statusStr, dragon.Memory, dragon.Soul)

	var chatMsgs []deepseek.Message
	chatMsgs = append(chatMsgs, deepseek.Message{Role: "system", Content: systemPrompt})
	for _, m := range history {
		role := "user"
		if m.Role == "dragon" {
			role = "assistant"
		}
		chatMsgs = append(chatMsgs, deepseek.Message{Role: role, Content: m.Content})
	}

	resp, err := dsClient.StreamChat(chatMsgs)
	return resp, &dragon, err
}

func (s *DragonService) SaveDragonReply(userID uint, dragonID uint, reply string) {
	dragonReply := model.DragonMessage{
		DragonID: dragonID,
		UserID:   userID,
		Role:     "dragon",
		Content:  reply,
	}
	global.DB.Create(&dragonReply)

	// 解析状态指令
	s.parseAndApplyStatus(dragonID, reply)

	// 检查演化
	var count int64
	global.DB.Model(&model.DragonMessage{}).Where("dragon_id = ?", dragonID).Count(&count)
	if count >= 20 {
		var d model.Dragon
		global.DB.First(&d, dragonID)
		go s.evolveDragonSoul(&d)
	}
}

// evolveDragonSoul 总结记忆并重塑灵魂
func (s *DragonService) evolveDragonSoul(dragon *model.Dragon) {
	fmt.Printf("[Soul Evolution] Dragon %s is evolving its soul...\n", dragon.Name)

	var msgs []model.DragonMessage
	global.DB.Where("dragon_id = ?", dragon.ID).Order("created_at asc").Find(&msgs)

	historyStr := ""
	for _, m := range msgs {
		historyStr += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}

	systemPrompt := `You are a soul architect for dragons. Analyze the conversation history and the current state.
Update the dragon's Memory and Soul.
- Memory: A concise summary of important events and interactions (under 100 words).
- Soul: A description of its personality, temperament, and current emotional state (under 50 words).

Output ONLY in JSON format: {"memory": "...", "soul": "..."}`

	userPrompt := fmt.Sprintf("Current Memory: %s\nCurrent Soul: %s\n\nNew Conversations:\n%s", dragon.Memory, dragon.Soul, historyStr)

	dsClient := deepseek.NewClient(global.CONFIG.GetString("deepseek.api_key"))
	aiMsgs := []deepseek.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := dsClient.Chat(aiMsgs)
	if err != nil {
		fmt.Printf("[Soul Evolution] Error: %v\n", err)
		return
	}

	var result struct {
		Memory string `json:"memory"`
		Soul   string `json:"soul"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// 尝试修复非 JSON 返回
		fmt.Printf("[Soul Evolution] JSON Parse Error: %v\n", err)
		return
	}

	// 更新龙的数据
	dragon.Memory = result.Memory
	dragon.Soul = result.Soul
	global.DB.Save(dragon)

	// 减少上下文：保留最近 10 条，删除旧消息
	var last10 []model.DragonMessage
	global.DB.Where("dragon_id = ?", dragon.ID).Order("created_at desc").Limit(10).Find(&last10)

	if len(last10) > 0 {
		oldestID := last10[len(last10)-1].ID
		global.DB.Where("dragon_id = ? AND id < ?", dragon.ID, oldestID).Unscoped().Delete(&model.DragonMessage{})
	}

	fmt.Printf("[Soul Evolution] Dragon %s soul evolved. New Soul: %s\n", dragon.Name, dragon.Soul)
}

func (s *DragonService) parseAndApplyStatus(dragonID uint, text string) {
	if !strings.Contains(text, "获得状态：") {
		return
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.Contains(line, "获得状态：") {
			parts := strings.Split(line, "获得状态：")
			if len(parts) > 1 {
				directive := strings.TrimSpace(parts[1])
				dParts := strings.Split(directive, "|")
				if len(dParts) >= 4 {
					name := strings.TrimSpace(dParts[0])
					duration, _ := strconv.Atoi(strings.TrimSpace(dParts[1]))
					effect := strings.TrimSpace(dParts[2])
					desc := strings.TrimSpace(dParts[3])

					if duration <= 0 {
						duration = 1
					}

					expiresAt := time.Now().Add(time.Duration(duration) * time.Hour)
					status := model.DragonStatus{
						DragonID:  dragonID,
						Name:      name,
						Desc:      desc,
						Effect:    effect,
						ExpiresAt: &expiresAt,
					}
					global.DB.Create(&status)
					fmt.Printf("[Dragon Status] Dragon %d acquired status: %s via event/chat\n", dragonID, name)
				}
			}
		}
	}
}
