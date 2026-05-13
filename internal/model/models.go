package model

import (
	"gorm.io/gorm"
	"time"
)

// Base 模型，替代 gorm.Model，将时间戳交给 MySQL 处理
type Base struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `gorm:"->;<-:create;type:timestamp;default:CURRENT_TIMESTAMP;index" json:"created_at"`
	UpdatedAt time.Time      `gorm:"->;type:timestamp;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	Base
	Username          string     `gorm:"uniqueIndex;type:varchar(100);not null" json:"username"`
	Password          string     `gorm:"type:varchar(255);not null" json:"-"`
	Avatar            string     `gorm:"type:varchar(255)" json:"avatar"`
	Email             string     `gorm:"type:varchar(100)" json:"email"`
	Phone             string     `gorm:"type:varchar(20)" json:"phone"`
	Role              string     `gorm:"type:varchar(20);default:'user'" json:"role"` // 'user', 'admin'
	Experience        int        `gorm:"default:0" json:"experience"`
	Title             string     `gorm:"type:varchar(100);default:'初入龙屿的游侠'" json:"title"`
	NicknameChangedAt *time.Time `json:"nickname_changed_at"`
}

type Message struct {
	Base
	Content          string   `gorm:"type:text;not null" json:"content"`
	UserID           *uint    `gorm:"index" json:"user_id"`
	User             User     `gorm:"foreignKey:UserID" json:"user"`
	IsAIReply        bool     `gorm:"default:false" json:"is_air_reply"`
	IsRecalled       bool     `gorm:"default:false" json:"is_recalled"`
	ReplyToMessageID *uint    `gorm:"index" json:"reply_to_id"`
	ReplyToMessage   *Message `gorm:"foreignKey:ReplyToMessageID" json:"reply_to_message"`
	AIInterest       bool     `gorm:"default:false" json:"ai_interest"`
	IsForceReplied   bool     `gorm:"default:false" json:"is_force_replied"`
}

type MagicRecord struct {
	Base
	UserID uint   `gorm:"index" json:"user_id"`
	Prompt string `gorm:"type:text" json:"prompt"`
}

type Archive struct {
	Base
	Title   string `gorm:"type:varchar(255);not null" json:"title"`
	Content string `gorm:"type:text" json:"content"`
	Date    string `gorm:"type:varchar(20);index" json:"date"`
	Type    int    `gorm:"default:0" json:"type"` // 0: 编年史, 1: 个人传记
}

type Feedback struct {
	Base
	UserID       uint   `json:"user_id"`
	Content      string `gorm:"type:text;not null" json:"content"`
	IsReplied    bool   `gorm:"default:false" json:"is_replied"`
	ReplyContent string `gorm:"type:text" json:"reply_content"`
}

type Dragon struct {
	Base
	UserID       uint       `gorm:"uniqueIndex" json:"user_id"`
	Name         string     `gorm:"type:varchar(100)" json:"name"`
	Rarity       string     `gorm:"type:varchar(20)" json:"rarity"` // 'common', 'rare', 'epic'
	Stage        int        `gorm:"default:0" json:"stage"`        // 0:蛋, 1:幼崽, 2:雏龙, 3:巨龙
	Hunger       int        `gorm:"default:50" json:"hunger"`
	MaxHunger    int        `gorm:"default:100" json:"max_hunger"`
	Happiness    int        `gorm:"default:50" json:"happiness"`
	MaxHappiness int        `gorm:"default:100" json:"max_happiness"`
	Exp          int        `gorm:"default:0" json:"exp"`
	LastFedAt    *time.Time `json:"last_fed_at"`
	LastPlayedAt *time.Time `json:"last_played_at"`
	ImageURL     string     `gorm:"type:varchar(255)" json:"image_url"`
	BasePrompt   string     `gorm:"type:text" json:"base_prompt"`
	Seed         int64      `json:"seed"`
	Personality  string     `gorm:"type:varchar(100)" json:"personality"` // 性格标签 (如: 傲娇, 憨厚)
	Memory       string     `gorm:"type:text" json:"memory"`              // 龙的长期记忆（由 AI 定期总结）
	Soul         string     `gorm:"type:text" json:"soul"`                // 龙 the 灵魂特质（性格、喜好、当前状态描述）
	IsGone       bool       `gorm:"default:false" json:"is_gone"`
	Statuses     []DragonStatus `gorm:"foreignKey:DragonID" json:"statuses"`
	DailyEventCount int        `gorm:"default:0" json:"daily_event_count"`
	LastEventAt     *time.Time `json:"last_event_at"`
}

type DragonStatus struct {
	Base
	DragonID  uint       `gorm:"index" json:"dragon_id"`
	Name      string     `gorm:"type:varchar(100)" json:"name"`
	Desc      string     `gorm:"type:varchar(255)" json:"desc"`
	Effect    string     `gorm:"type:varchar(100)" json:"effect"` // 格式: hunger-10, happiness+5 等
	ExpiresAt *time.Time `json:"expires_at"`
}

type DragonMessage struct {
	Base
	DragonID uint   `gorm:"index" json:"dragon_id"`
	UserID   uint   `gorm:"index" json:"user_id"`
	Role     string `gorm:"type:varchar(20)" json:"role"` // 'user' 或 'dragon'
	Content  string `gorm:"type:text" json:"content"`
}

type UserItem struct {
	Base
	UserID uint   `gorm:"index" json:"user_id"`
	Type   string `gorm:"type:varchar(50)" json:"type"` // 'food', 'toy'
	Count  int    `gorm:"default:0" json:"count"`
}

type UserTask struct {
	Base
	UserID      uint   `gorm:"index" json:"user_id"`
	TaskType    string `gorm:"type:varchar(50)" json:"task_type"` // 'sign_in', 'chat', 'generate', 'share'
	Progress    int    `gorm:"default:0" json:"progress"`
	MaxProgress int    `gorm:"default:1" json:"max_progress"`
	Date        string `gorm:"type:varchar(20);index" json:"date"` // 格式: 2024-05-13
	IsClaimed   bool   `gorm:"default:false" json:"is_claimed"`
}

type FortuneRecord struct {
	Base
	UserID         uint   `gorm:"index" json:"user_id"`
	Luck           string `gorm:"type:varchar(50)" json:"luck"`         // 大吉, 小吉...
	Verse          string `gorm:"type:varchar(255)" json:"verse"`       // 签头 (4字)
	Interpretation string `gorm:"type:text" json:"interpretation"`      // 详细神谕
	Suit           string `gorm:"type:varchar(255)" json:"suit"`        // 宜 (逗号分隔)
	Avoid          string `gorm:"type:varchar(255)" json:"avoid"`       // 忌 (逗号分隔)
	Date           string `gorm:"type:varchar(20);index" json:"date"`
}
