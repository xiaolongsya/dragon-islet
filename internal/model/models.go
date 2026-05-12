package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username          string    `gorm:"type:varchar(50)" json:"username"`
	Phone             string    `gorm:"uniqueIndex;type:varchar(20);not null" json:"phone"`
	Password          string    `gorm:"type:varchar(255)" json:"-"`
	Avatar            string    `gorm:"type:varchar(255)" json:"avatar"`
	Role              string    `gorm:"type:varchar(20);default:'user'" json:"role"` // 'admin', 'user', 'ai'
	Motto             string    `gorm:"type:varchar(255)" json:"motto"`                       // 新增：游侠宣言
	Title             string    `gorm:"type:varchar(50);default:'初入龙屿的游侠'" json:"title"`      // 新增：身份称号
	Experience        int       `gorm:"default:0" json:"experience"`                          // 新增：灵力值
	NicknameChangedAt *time.Time `json:"nickname_changed_at"`
}

type Message struct {
	gorm.Model
	Content          string   `gorm:"type:text;not null" json:"content"`
	UserID           *uint    `json:"user_id"`
	User             User     `gorm:"foreignKey:UserID" json:"user"`
	IsAIReply        bool     `gorm:"default:false" json:"is_ai_reply"`
	AIInterest       bool     `gorm:"default:false" json:"ai_interest"` // 新增：龙主兴趣状态
	IsRecalled       bool     `gorm:"default:false" json:"is_recalled"` // 新增：撤回状态
	IsForceReplied   bool     `gorm:"default:false" json:"is_force_replied"` // 新增：是否使用了秘宝
	ReplyToMessageID *uint    `json:"reply_to_message_id"`
	ReplyToMessage   *Message `gorm:"foreignKey:ReplyToMessageID" json:"reply_to_message"`
}

func (m *Message) TableName() string {
	return "messages"
}

type Archive struct {
	gorm.Model
	Title   string `gorm:"type:varchar(255)" json:"title"`
	Content string `gorm:"type:longtext" json:"content"` // AI 生成的或手动录入的内容
	Date    string `gorm:"type:varchar(20)" json:"date"`  // 格式：2024-05-09
	Type    int    `gorm:"default:0" json:"type"`        // 0: 每日行纪, 1: 铸龙图谱
}

// MagicRecord 龙语显像记录表
type MagicRecord struct {
	gorm.Model
	UserID   uint   `json:"user_id"`
	ImageURL string `json:"image_url"`
}

type Feedback struct {
	gorm.Model
	UserID       uint   `json:"user_id"`
	Content      string `gorm:"type:text;not null" json:"content"`
	IsReplied    bool   `gorm:"default:false" json:"is_replied"`
	ReplyContent string `gorm:"type:text" json:"reply_content"`
}
