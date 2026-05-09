package deepseek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	ApiKey string
	BaseUrl string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error interface{} `json:"error"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		ApiKey: apiKey,
		BaseUrl: "https://api.deepseek.com", // 示例地址，具体以官方为准
	}
}

func (c *Client) Chat(messages []Message) (string, error) {
	reqBody := ChatRequest{
		Model:    "deepseek-chat",
		Messages: messages,
	}
	
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", c.BaseUrl+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", err
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("deepseek error: %v", chatResp.Error)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from deepseek")
}
