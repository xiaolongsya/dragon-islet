package initialize

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() {
	dsn := global.CONFIG.GetString("db.dsn")
	if dsn == "" {
		// 默认本地开发环境
		dsn = "root:123456@tcp(127.0.0.1:3306)/dragon_islet?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 强制重置求签表以应用最新精简结构
	db.Migrator().DropTable(&model.FortuneRecord{})

	// 自动迁移模型
	err = db.AutoMigrate(
		&model.User{},
		&model.Message{},
		&model.MagicRecord{},
		&model.Archive{},
		&model.Feedback{},
		&model.Dragon{},
		&model.UserItem{},
		&model.UserTask{},
		&model.FortuneRecord{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}

	global.DB = db
	fmt.Println("Database initialized successfully.")
}
