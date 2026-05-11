package service

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
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
