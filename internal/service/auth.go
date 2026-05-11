package service

import (
	"context"
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"dragon-islet/pkg/aliyun"
	"dragon-islet/pkg/utils"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct{}

// Register 用户名密码注册 (加入验证码校验)
func (s *AuthService) Register(username, password, phone, code string) error {
	ctx := context.Background()

	// 1. 校验验证码
	codeKey := fmt.Sprintf("sms_code:%s", phone)
	savedCode, err := global.REDIS.Get(ctx, codeKey).Result()
	if err != nil || savedCode != code {
		return errors.New("验证码错误或已过期")
	}

	// 2. 检查用户名是否存在
	var count int64
	global.DB.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return errors.New("该名号已被其他游侠占用")
	}

	// 3. 检查手机号是否已注册
	global.DB.Model(&model.User{}).Where("phone = ?", phone).Count(&count)
	if count > 0 {
		return errors.New("该手机号已绑定了其他游侠")
	}

	// 加密密码
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user := model.User{
		Username: username,
		Password: string(hashedPassword),
		Phone:    phone,
		Avatar:   "https://xiaolongya.cn/uploads/1778432333617872906.jpg",
		Role:     "user",
	}

	if err := global.DB.Create(&user).Error; err != nil {
		return err
	}

	// 4. 注册成功删除验证码
	global.REDIS.Del(ctx, codeKey)
	return nil
}

// Login 用户名密码登录
func (s *AuthService) Login(username, password string) (string, *model.User, error) {
	var user model.User
	if err := global.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, errors.New("游侠未被记载（用户不存在）")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("密语错误（密码错误）")
	}

	token, _ := utils.GenerateToken(user.ID, user.Role)
	return token, &user, nil
}

// SendSms 发送验证码 (用于找回密码)
func (s *AuthService) SendSms(phone string) error {
	ctx := context.Background()
	limitKey := fmt.Sprintf("sms_limit:%s", phone)
	if exists, _ := global.REDIS.Exists(ctx, limitKey).Result(); exists > 0 {
		return errors.New("发送太频繁，请1分钟后再试")
	}

	smsClient := aliyun.NewSmsClient()
	code, err := smsClient.SendVerifyCode(phone)
	if err != nil {
		return err
	}

	global.REDIS.Set(ctx, fmt.Sprintf("sms_code:%s", phone), code, 5*time.Minute)
	global.REDIS.Set(ctx, limitKey, "1", 1*time.Minute)
	return nil
}

// ResetPassword 通过验证码重置密码
func (s *AuthService) ResetPassword(phone, code, newPassword string) error {
	ctx := context.Background()
	// 1. 校验验证码
	savedCode, err := global.REDIS.Get(ctx, fmt.Sprintf("sms_code:%s", phone)).Result()
	if err != nil || savedCode != code {
		return errors.New("验证码错误或已过期")
	}

	// 2. 更新用户密码
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	res := global.DB.Model(&model.User{}).Where("phone = ?", phone).Update("password", string(hashedPassword))
	if res.RowsAffected == 0 {
		return errors.New("该手机号未绑定任何游侠")
	}

	global.REDIS.Del(ctx, fmt.Sprintf("sms_code:%s", phone))
	return nil
}

// UpdateProfile 修改昵称和头像
func (s *AuthService) UpdateProfile(userID uint, nickname, avatar string) error {
	var user model.User
	global.DB.First(&user, userID)

	updates := make(map[string]interface{})

	// 昵称修改限流 (24小时一次)
	if nickname != "" && nickname != user.Username {
		if user.NicknameChangedAt != nil && time.Since(*user.NicknameChangedAt).Hours() < 24 {
			return errors.New("名号改动太频繁，需等待一个昼夜（24小时）")
		}
		// 查重：排除自身
		var taken int64
		global.DB.Model(&model.User{}).Where("username = ? AND id != ?", nickname, userID).Count(&taken)
		if taken > 0 {
			return errors.New("该名号已被其他游侠占用")
		}
		updates["username"] = nickname
		now := time.Now()
		updates["nickname_changed_at"] = &now
	}

	if avatar != "" {
		updates["avatar"] = avatar
	}

	return global.DB.Model(&user).Updates(updates).Error
}

// UpdatePasswordInternal 修改密码 (校验旧手机验证码)
func (s *AuthService) UpdatePasswordInternal(userID uint, code, newPassword string) error {
	var user model.User
	global.DB.First(&user, userID)
	return s.ResetPassword(user.Phone, code, newPassword)
}
