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
	Model            string `json:"model"`
	Prompt           string `json:"prompt"`
	Size             string `json:"size"`
	Resolution       string `json:"resolution"`
	Quality          string `json:"quality"`
	N                int    `json:"n"`
	OfficialFallback bool   `json:"official_fallback"`
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

// CreateImage 提交生成任务
func (c *Client) CreateImage(prompt string, size string) (string, error) {
	url := fmt.Sprintf("%s/images/generations", c.BaseURL)
	if size == "" {
		size = "1:1"
	}
	payload := ImageRequest{
		Model:            "gpt-image-2",
		Prompt:           prompt,
		Size:             size,
		Resolution:       "1k",
		Quality:          "low",
		N:                1,
		OfficialFallback: true,
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
