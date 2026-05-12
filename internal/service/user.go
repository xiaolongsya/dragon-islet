package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/apimart"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type UserService struct{}

var luckRanks = []string{"✨ 大吉", "☀️ 小吉", "⛅ 中平", "🌧️ 下签", "🌑 凶"}

func (s *UserService) GetDailyFortune(userID uint) (*model.FortuneRecord, error) {
	todayStr := time.Now().Format("2006-01-02")
	
	var record model.FortuneRecord
	err := global.DB.Where("user_id = ? AND date = ?", userID, todayStr).First(&record).Error
	if err == nil {
		return &record, nil
	}

	rand.Seed(time.Now().UnixNano())
	baseLuck := luckRanks[rand.Intn(len(luckRanks))]

	token := global.CONFIG.GetString("APIMART_TOKEN")
	client := apimart.NewClient(token)
	
	systemPrompt := `你现在是龙屿的'司命祭司'。
请严格按照以下 JSON 格式回复，不要包含任何额外文字：
{
  "luck": "吉凶评级 (如: ✨ 大吉, ☀️ 小吉, ⛅ 中平, 🌧️ 下签, 🌑 凶)",
  "verse": "四字签头 (如: 潜龙出渊)",
  "interpretation": "50-100字的神谕，必须给出今日的具体行动建议",
  "suit": ["动作1", "动作2", "动作3"],
  "avoid": ["动作1", "动作2", "动作3"]
}
要求：suit 和 avoid 必须各包含 3 个具体的动作（现实生活+龙屿背景结合）。`

	userPrompt := fmt.Sprintf("今日运势基调：【%s】。请为游侠降下神谕。", baseLuck)
	resp, err := client.GetChatCompletion("gpt-4o-mini", userPrompt, systemPrompt)
	
	cleanJSON := strings.TrimSpace(resp)
	cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	cleanJSON = strings.TrimSpace(cleanJSON)

	var aiResult struct {
		Luck           string   `json:"luck"`
		Verse          string   `json:"verse"`
		Interpretation string   `json:"interpretation"`
		Suit           []string `json:"suit"`
		Avoid          []string `json:"avoid"`
	}

	if err != nil || json.Unmarshal([]byte(cleanJSON), &aiResult) != nil {
		aiResult.Luck = baseLuck
		aiResult.Verse = "灵力波涌"
		aiResult.Interpretation = "星轨在迷雾中交织，神龙的意志难以捉摸。今日宜保持敬畏，静候契机。"
		aiResult.Suit = []string{"静坐调息", "擦拭逆鳞", "翻阅古籍"}
		aiResult.Avoid = []string{"轻许诺言", "妄动禁忌", "独闯深渊"}
	}

	record = model.FortuneRecord{
		UserID:         userID,
		Luck:           aiResult.Luck,
		Verse:          aiResult.Verse,
		Interpretation: aiResult.Interpretation,
		Suit:           strings.Join(aiResult.Suit, ","),
		Avoid:          strings.Join(aiResult.Avoid, ","),
		Date:           todayStr,
	}

	if err := global.DB.Create(&record).Error; err != nil {
		return nil, err
	}

	return &record, nil
}
