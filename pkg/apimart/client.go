package apimart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Client struct {
	Token   string
	BaseURL string
}

func NewClient(token string) *Client {
	return &Client{
		Token:   token,
		BaseURL: "https://api.apimart.ai/v1",
	}
}

type ImageRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	Size             string   `json:"size"`
	Resolution       string   `json:"resolution"`
	Quality          string   `json:"quality"`
	N                int      `json:"n"`
	OfficialFallback bool     `json:"official_fallback"`
	ImageURLs        []string `json:"image_urls,omitempty"`
}

type TaskResponse struct {
	Code int `json:"code"`
	Data []struct {
		Status string `json:"status"`
		TaskID string `json:"task_id"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type ResultResponse struct {
	Code int `json:"code"`
	Data struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Progress int    `json:"progress"`
		Result   struct {
			Images []struct {
				URL []string `json:"url"`
			} `json:"images"`
		} `json:"result"`
	} `json:"data"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CreateImage 提交生成任务
func (c *Client) CreateImage(prompt string, size string, resolution string, imageURLs []string) (string, error) {
	url := fmt.Sprintf("%s/images/generations", c.BaseURL)
	if size == "" {
		size = "1:1"
	}
	if resolution == "" {
		resolution = "1k"
	}
	payload := ImageRequest{
		Model:            "gpt-image-2",
		Prompt:           prompt,
		Size:             size,
		Resolution:       resolution,
		N:                1,
		OfficialFallback: true,
		ImageURLs:        imageURLs,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tr TaskResponse
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", err
	}

	if tr.Code != 200 {
		if tr.Error != nil {
			return "", errors.New(tr.Error.Message)
		}
		return "", fmt.Errorf("API Error: %d", tr.Code)
	}

	if len(tr.Data) > 0 {
		return tr.Data[0].TaskID, nil
	}
	return "", errors.New("未获取到任务ID")
}

// GetTaskStatus 查询任务进度和结果
func (c *Client) GetTaskStatus(taskID string) (*ResultResponse, error) {
	url := fmt.Sprintf("%s/tasks/%s", c.BaseURL, taskID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rr ResultResponse
	body, _ := ioutil.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &rr); err != nil {
		return nil, err
	}

	return &rr, nil
}

// GetChatCompletionWithHistory 获取带历史背景的 AI 回复
func (c *Client) GetChatCompletionWithHistory(model string, messages []ChatMessage) (string, error) {
	url := fmt.Sprintf("%s/chat/completions", c.BaseURL)
	payload := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: 0.7,
		Stream:      false,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API 灵力不稳 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var cr ChatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("神谕解析失败 (JSON Error): %v | Body: %s", err, string(body))
	}

	if len(cr.Choices) > 0 {
		return cr.Choices[0].Message.Content, nil
	}
	return "", errors.New("AI 未能降下神启")
}

// GetChatCompletion 获取 AI 文本回复 (单次)
func (c *Client) GetChatCompletion(model string, prompt string, systemPrompt string) (string, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}
	return c.GetChatCompletionWithHistory(model, messages)
}
