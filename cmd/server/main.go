package main

import (
	"dragon-islet/internal/handler"
	"dragon-islet/internal/initialize"
	"net/http"

	"dragon-islet/internal/logic"
	"dragon-islet/internal/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	initialize.InitConfig()
	initialize.InitDB()
	initialize.InitRedis()

	// 启动 WebSocket 枢纽
	go logic.GlobalHub.Run()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	chatHandler := &handler.ChatHandler{}
	authHandler := &handler.AuthHandler{}
	feedbackHandler := &handler.FeedbackHandler{}

	api := r.Group("/dragon")
	{
		// WebSocket 入口
		api.GET("/ws", chatHandler.WsChat)

		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/send-sms", authHandler.SendSms)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		// 史诗相关
		archiveHandler := &handler.ArchiveHandler{}
		api.GET("/archives", archiveHandler.List)
		api.POST("/archives/generate", archiveHandler.ManualGenerate)

		// 龙主语录 (公开)
		quoteHandler := &handler.QuoteHandler{}
		api.GET("/quote", quoteHandler.Get)

		// 聊天列表 (公开)
		api.GET("/chat/list", chatHandler.List)

		// 图片上传 (需要 JWT)
		uploadHandler := &handler.UploadHandler{}
		api.POST("/upload", middleware.JWTAuth(), uploadHandler.UploadImage)

		// 聊天发言 (需要 JWT)
		chatAuth := api.Group("/chat")
		chatAuth.Use(middleware.JWTAuth())
		{
			chatAuth.POST("/send", chatHandler.Send)
			chatAuth.DELETE("/:id", chatHandler.Delete)
			chatAuth.GET("/my", chatHandler.MyMessages)
		}

		// 用户相关 (需要 JWT)
		userHandler := &handler.UserHandler{}
		userGroup := api.Group("/user")
		userGroup.Use(middleware.JWTAuth())
		{
			userGroup.POST("/profile", userHandler.UpdateProfile)
			userGroup.POST("/password", userHandler.UpdatePassword)
		}

		// 匿名信
		feedbackGroup := api.Group("/feedback")
		feedbackGroup.Use(middleware.JWTAuth())
		{
			feedbackGroup.POST("/submit", feedbackHandler.Submit)
			feedbackGroup.GET("/my", feedbackHandler.List)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 静态文件服务
	// 静态文件访问交由 Nginx 处理，服务器不再直接暴露 uploads 目录。
	// 如需在 Go 中启用静态服务，可取消下面代码并确保 save_path 为正确绝对路径。
	// saveDir := global.CONFIG.GetString("upload.save_path")
	// if saveDir == "" {
	//         saveDir = "./uploads"
	// }
	// r.Static("/dragon/uploads", saveDir)

	r.Run(":8888")
}
