package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"errors"
)

type FeedbackService struct{}

func (s *FeedbackService) CreateFeedback(userID uint, content string) error {
	fb := model.Feedback{
		UserID:  userID,
		Content: content,
	}
	return global.DB.Create(&fb).Error
}

func (s *FeedbackService) GetUserFeedbacks(userID uint) ([]model.Feedback, error) {
	var feedbacks []model.Feedback
	err := global.DB.Where("user_id = ?", userID).Order("created_at desc").Find(&feedbacks).Error
	return feedbacks, err
}

func (s *FeedbackService) DeleteFeedback(userID uint, fbID uint) error {
	var fb model.Feedback
	if err := global.DB.Where("id = ? AND user_id = ?", fbID, userID).First(&fb).Error; err != nil {
		return err
	}
	if fb.IsReplied {
		return errors.New("已收到回响，无法抹除")
	}
	return global.DB.Delete(&fb).Error
}

func (s *FeedbackService) AdminGetList(page, limit int) ([]model.Feedback, int64, error) {
	var feedbacks []model.Feedback
	var total int64
	global.DB.Model(&model.Feedback{}).Count(&total)
	err := global.DB.Order("created_at desc").Offset((page - 1) * limit).Limit(limit).Find(&feedbacks).Error
	return feedbacks, total, err
}

func (s *FeedbackService) AdminReply(fbID uint, replyContent string) error {
	return global.DB.Model(&model.Feedback{}).Where("id = ?", fbID).Updates(map[string]interface{}{
		"reply_content": replyContent,
		"is_replied":    true,
	}).Error
}
