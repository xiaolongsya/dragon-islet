package service

import (
	"bytes"
	"crypto/md5"
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type TTSService struct{}

const (
	wsURL = "wss://dashscope.aliyuncs.com/api-ws/v1/inference/"
)

// GetDragonVoice 获取龙的语音文件路径
func (s *TTSService) GetDragonVoice(dragon *model.Dragon, text string) (string, error) {
	// 1. 根据阶段选择基础音色和指令
	voice, instruction := s.GetVoiceAndInstruction(dragon)
	
	// 2. 检查缓存
	hash := md5.Sum([]byte(text + instruction + voice))
	fileName := hex.EncodeToString(hash[:]) + ".mp3"
	audioDir := filepath.Join(global.CONFIG.GetString("upload_path"), "audio")
	filePath := filepath.Join(audioDir, fileName)
	
	if _, err := os.Stat(filePath); err == nil {
		return fileName, nil
	}

	os.MkdirAll(audioDir, 0755)

	// 3. 调用阿里云合成
	err := s.synthesize(text, voice, instruction, filePath)
	if err != nil {
		return "", err
	}

	return fileName, nil
}

// GetVoiceAndInstruction 根据龙的阶段返回最贴合的音色和指令
func (s *TTSService) GetVoiceAndInstruction(dragon *model.Dragon) (string, string) {
	voice := "longanyang" // 锁死已被验证通过的标准底音
	var stageDesc string

	switch dragon.Stage {
	case 0: // 龙蛋
		stageDesc = "底音大幅上扬模拟 3 岁幼儿，奶声奶气，极其稚嫩，语速缓慢且带有一丝空灵梦幻感。"
	case 1: // 幼龙
		stageDesc = "底音上扬模拟 6 岁男童，清脆活泼，语气天真无邪，充满了对世界的好奇心，语速较快。"
	case 2: // 青年龙
		stageDesc = "标准少年音，声音清朗、干净、充满朝气，语气爽朗自信，充满活力。"
	case 3: // 壮年龙
		stageDesc = "底音下沉，表现出成熟稳重的中年男性质感，语速从容平稳，声音雄浑有力。"
	case 4: // 真龙
		stageDesc = "极致压抑的重低音烟嗓，底音降至最低，带有一种神圣且不可侵犯的皇室威压，语速极慢。"
	default:
		stageDesc = "声音平稳自然。"
	}

	personalityDesc := ""
	p := dragon.Personality
	if strings.Contains(p, "傲娇") {
		personalityDesc = "语气带着一丝傲慢与不屑，显得很有性格。"
	} else if strings.Contains(p, "温顺") || strings.Contains(p, "呆萌") {
		personalityDesc = "语气温柔乖巧，带有明显的治愈感。"
	} else if strings.Contains(p, "调皮") {
		personalityDesc = "语气灵动跳跃，显得古灵精怪。"
	}

	return voice, fmt.Sprintf("%s %s", stageDesc, personalityDesc)
}

func (s *TTSService) synthesize(text, voice, instruction, outputPath string) error {
	apiKey := global.CONFIG.GetString("aliyun.api_key")
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("APIMART_TOKEN") 
	}

	if apiKey == "" {
		fmt.Println("[TTS] Error: DASHSCOPE_API_KEY / APIMART_TOKEN is empty. Please fill it in .env and restart.")
		return fmt.Errorf("api key is empty")
	}

	url := "https://dashscope.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer"
	
	// 根据文档对齐 (Line 387-394 & 509)
	// 1. voice, format, sample_rate 必须在 input 结构中
	// 2. instructions (复数) 在 parameters 结构中
	reqBody := map[string]interface{}{
		"model": "cosyvoice-v3-flash",
		"input": map[string]interface{}{
			"text":        text,
			"voice":       voice, // 使用动态音色
			"format":      "mp3",
			"sample_rate": 22050,
		},
		"parameters": map[string]interface{}{
			"instructions": instruction, // 注意这里是复数 instructions
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	fmt.Printf("[TTS] Calling Aliyun HTTP API (CosyVoice v3-flash). Text: %s\n", text)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[TTS] HTTP Request failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("[TTS] API Response: Status=%d, Content-Type=%s\n", resp.StatusCode, contentType)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[TTS] API Error Response: %s\n", string(body))
		return fmt.Errorf("aliyun api error: status %d, body %s", resp.StatusCode, string(body))
	}

	// 如果返回的是音频流
	if strings.Contains(contentType, "audio/") {
		fmt.Println("[TTS] Received direct audio stream. Saving...")
		file, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, resp.Body)
		return err
	}

	// 如果返回的是 JSON (通常包含音频 URL)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Output struct {
			Audio struct {
				URL string `json:"url"`
			} `json:"audio"`
			TaskID     string `json:"task_id"`
			TaskStatus string `json:"task_status"`
		} `json:"output"`
		RequestID string `json:"request_id"`
		Message   string `json:"message"`
		Code      string `json:"code"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("[TTS] JSON Parse failed: %v\n, Body: %s", err, string(body))
		return err
	}

	if result.Code != "" && result.Code != "200" {
		return fmt.Errorf("api error code: %s, msg: %s", result.Code, result.Message)
	}

	audioURL := result.Output.Audio.URL
	if audioURL == "" {
		// 如果返回了 task_id，说明是异步任务，当前逻辑不支持异步轮询，需报错
		if result.Output.TaskID != "" {
			fmt.Printf("[TTS] Received TaskID: %s. This is an asynchronous task.\n", result.Output.TaskID)
			return fmt.Errorf("asynchronous task not supported yet, task_id: %s", result.Output.TaskID)
		}
		fmt.Printf("[TTS] No URL found. Full Response: %s\n", string(body))
		return fmt.Errorf("no audio url in response")
	}

	fmt.Printf("[TTS] Downloading audio from: %s\n", audioURL)
	audioResp, err := http.Get(audioURL)
	if err != nil {
		return err
	}
	defer audioResp.Body.Close()

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, audioResp.Body)
	if err == nil {
		fmt.Printf("[TTS] Audio saved to: %s\n", outputPath)
	}
	return err
}
