package main

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/handler"
	"dragon-islet/internal/initialize"
	"net/http"

	"dragon-islet/internal/logic"
	"dragon-islet/internal/middleware"
	"dragon-islet/internal/service"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	initialize.InitConfig()
	initialize.InitDB()
	initialize.InitRedis()

	// 启动 WebSocket 枢纽
	go logic.GlobalHub.Run()

	// 启动定时任务 (每日 03:00 生成史诗)
	c := cron.New()
	archiveService := &service.ArchiveService{}
	c.AddFunc("0 3 * * *", func() {
		fmt.Println("[Cron] 开始生成每日史诗...")
		if err := archiveService.GenerateDailyArchive(); err != nil {
			fmt.Printf("[Cron] 生成史诗失败: %v\n", err)
		} else {
			fmt.Println("[Cron] 每日史诗生成成功")
		}
	})
	c.Start()

	// 启动龙嗣生命周期管理 (饱食度/心情自动扣除)
	dragonService := &service.DragonService{}
	dragonService.StartLifeCycle()

	r := gin.Default()

	// 初始化监控
	initialize.InitPrometheus(r)

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

	// 初始化 Service
	authSvc := service.AuthService{}
	chatSvc := service.ChatService{}
	dragonSvc := service.DragonService{}
	feedbackSvc := service.FeedbackService{}

	// 启动龙嗣生命周期管理
	dragonSvc.StartLifeCycle()

	// 初始化 Handler 并注入 Service
	chatHandler := handler.NewChatHandler(chatSvc, dragonSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(authSvc, service.UserService{}, dragonSvc)
	feedbackHandler := handler.NewFeedbackHandler(feedbackSvc)
	archiveHandler := handler.NewArchiveHandler(service.ArchiveService{})
	quoteHandler := &handler.QuoteHandler{}
	uploadHandler := &handler.UploadHandler{}

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
		api.GET("/archives", archiveHandler.List)
		api.GET("/archives/manifesto", archiveHandler.GetManifesto)
		api.POST("/archives/analyze", middleware.JWTAuth(), archiveHandler.Analyze)
		api.POST("/archives", middleware.JWTAuth(), archiveHandler.Create)
		api.POST("/archives/generate", middleware.JWTAuth(), archiveHandler.ManualGenerate)

		// 龙主语录 (公开)
		api.GET("/quote", quoteHandler.Get)

		// 聊天列表 (公开)
		api.GET("/chat/list", chatHandler.List)

		// 图片上传 (需要 JWT)
		api.POST("/upload", middleware.JWTAuth(), uploadHandler.UploadImage)

		// 聊天发言 (需要 JWT)
		chatAuth := api.Group("/chat")
		chatAuth.Use(middleware.JWTAuth())
		{
			chatAuth.POST("/send", chatHandler.Send)
			chatAuth.DELETE("/:id", chatHandler.Delete)
			chatAuth.GET("/my", chatHandler.MyMessages)
			chatAuth.POST("/force-reply", chatHandler.ForceReply)
			chatAuth.POST("/generate-image", chatHandler.GenerateImage)
		}

		// 用户相关 (需要 JWT)
		userGroup := api.Group("/user")
		userGroup.Use(middleware.JWTAuth())
		{
			userGroup.GET("/profile", userHandler.GetProfile)
			userGroup.POST("/profile", userHandler.UpdateProfile)
			userGroup.POST("/password", userHandler.UpdatePassword)
			userGroup.GET("/fortune", userHandler.GetFortune)
		}

		// 匿名信
		feedbackGroup := api.Group("/feedback")
		feedbackGroup.Use(middleware.JWTAuth())
		{
			feedbackGroup.POST("/submit", feedbackHandler.Submit)
			feedbackGroup.GET("/my", feedbackHandler.List)
			feedbackGroup.DELETE("/:id", feedbackHandler.Delete)
		}

		// 养成系统
		ttsSvc := service.TTSService{}
		dragonHandler := handler.NewDragonHandler(dragonSvc, chatSvc, ttsSvc)
		dragonGroup := api.Group("/raising")
		dragonGroup.Use(middleware.JWTAuth())
		{
			dragonGroup.GET("/status", dragonHandler.GetStatus)
			dragonGroup.GET("/summary", dragonHandler.GetSummary)
			dragonGroup.POST("/feed", dragonHandler.Feed)
			dragonGroup.POST("/play", dragonHandler.Play)
			dragonGroup.POST("/rename", dragonHandler.Rename)
			dragonGroup.POST("/generate-image", dragonHandler.GenerateImage)
			dragonGroup.POST("/share", dragonHandler.Share)
			dragonGroup.GET("/tasks", dragonHandler.GetTasks)
			dragonGroup.POST("/claim-reward", dragonHandler.ClaimReward)
			dragonGroup.POST("/use-item", dragonHandler.UseItem)
			dragonGroup.POST("/evolve", dragonHandler.Evolve)
			dragonGroup.GET("/chat", dragonHandler.GetChatHistory)
			dragonGroup.POST("/chat", dragonHandler.Chat)
			dragonGroup.POST("/release", dragonHandler.Release)
			dragonGroup.GET("/speak", dragonHandler.Speak)
		}

		// 管理员相关
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(), middleware.AdminAuth())
		{
			admin.GET("/feedback", feedbackHandler.AdminList)
			admin.POST("/feedback/reply", feedbackHandler.AdminReply)
			admin.POST("/manifesto/update", archiveHandler.UpdateManifesto)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 静态文件服务
	// 无论是在本地还是云端，如果本地存有文件，则开启静态访问
	saveDir := global.CONFIG.GetString("upload.save_path")
	if saveDir == "" {
		saveDir = "./uploads"
	}
	// 将 /uploads 映射到本地目录
	r.Static("/uploads", saveDir)
	fmt.Printf("[System] Static file service enabled: /uploads -> %s\n", saveDir)

	r.Run(":8888")
}
