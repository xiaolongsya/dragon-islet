package aliyun

import (
	"dragon-islet/internal/global"
	"encoding/json"
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/dypnsapi"
)

type SmsClient struct {
	AccessKeyId     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
}

func NewSmsClient() *SmsClient {
	return &SmsClient{
		AccessKeyId:     global.CONFIG.GetString("aliyun.access_key_id"),
		AccessKeySecret: global.CONFIG.GetString("aliyun.access_key_secret"),
		SignName:        global.CONFIG.GetString("aliyun.sign_name"),
		TemplateCode:    global.CONFIG.GetString("aliyun.template_code"),
	}
}

// SendVerifyCode 发送验证码，并返回生成的验证码用于后端校验
func (c *SmsClient) SendVerifyCode(phone string) (string, error) {
	client, err := dypnsapi.NewClientWithAccessKey("cn-hangzhou", c.AccessKeyId, c.AccessKeySecret)
	if err != nil {
		return "", err
	}

	request := dypnsapi.CreateSendSmsVerifyCodeRequest()
	request.Scheme = "https"
	request.PhoneNumber = phone
	request.SignName = c.SignName
	request.TemplateCode = c.TemplateCode
	
	// 设置验证码规则
	// 注意：如果你的模板里有 ${min} 变量，必须在这里补全。
	// 这里我们按照 verify.md 中的示例，同时传入 code 和 min
	param := map[string]string{
		"code": "##code##",
		"min":  "5", 
	}
	paramJson, _ := json.Marshal(param)
	request.TemplateParam = string(paramJson)
	request.ReturnVerifyCode = "true" // 关键：让接口直接返回生成的验证码

	response, err := client.SendSmsVerifyCode(request)
	if err != nil {
		return "", err
	}

	if response.Code != "OK" {
		return "", fmt.Errorf("aliyun error: %s", response.Message)
	}

	return response.Model.VerifyCode, nil
}
