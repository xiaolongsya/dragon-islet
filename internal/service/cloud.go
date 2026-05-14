package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CloudService struct{}

// UploadFileToCloud 将本地文件或远程URL上传至云端统一接口
func (s *CloudService) UploadFileToCloud(pathOrURL string, isRemote bool, userToken string) (string, error) {
	var data []byte
	var err error
	var filename string

	if isRemote {
		resp, err := http.Get(pathOrURL)
		if err != nil {
			return "", fmt.Errorf("failed to fetch remote file: %v", err)
		}
		defer resp.Body.Close()
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		filename = fmt.Sprintf("cloud_%d", time.Now().UnixNano())
		if strings.Contains(pathOrURL, ".mp3") {
			filename += ".mp3"
		} else {
			filename += ".jpg"
		}
	} else {
		data, err = os.ReadFile(pathOrURL)
		if err != nil {
			return "", fmt.Errorf("failed to read local file: %v", err)
		}
		filename = filepath.Base(pathOrURL)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	writer.Close()

	// 云端统一接口 (根据用户要求)
	uploadURL := "http://xiaolongya.cn:8888/dragon/upload"
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// 注入令牌
	if userToken != "" {
		if !strings.HasPrefix(userToken, "Bearer ") {
			userToken = "Bearer " + userToken
		}
		req.Header.Set("Authorization", userToken)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cloud request failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode cloud response: %v", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("cloud upload error: %s (code %d)", result.Msg, result.Code)
	}

	return result.Data.URL, nil
}
