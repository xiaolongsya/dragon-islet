package initialize

import (
	"dragon-islet/internal/global"
	"dragon-islet/internal/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

var (
	// 定义自定义业务指标
	dragonTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dragon_total_count",
		Help: "Total number of dragons in the islet",
	})
	
	activeUserGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dragon_active_users_24h",
		Help: "Number of active users in the last 24 hours",
	})

	messageTotalGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "dragon_messages_total_count",
		Help: "Total number of human messages across all channels (Square + Raising)",
	})
)

// InitPrometheus 初始化监控
func InitPrometheus(r *gin.Engine) {
	// 1. 设置基础中间件 (监控请求次数、耗时等)
	p := ginprometheus.NewPrometheus("gin")
	p.MetricsPath = "/dragon/metrics" // 只有管理员或 Prometheus 可访问的路径
	p.Use(r)

	// 2. 启动定时任务更新业务指标
	go func() {
		for {
			updateBusinessMetrics()
			time.Sleep(1 * time.Minute)
		}
	}()
}

func updateBusinessMetrics() {
	if global.DB == nil {
		return
	}

	// 统计龙的总数
	var dragonCount int64
	global.DB.Model(&model.Dragon{}).Where("is_gone = ?", false).Count(&dragonCount)
	dragonTotalGauge.Set(float64(dragonCount))

	// 统计 24 小时活跃用户 (根据最后一次发送消息的时间)
	var activeUsers int64
	last24h := time.Now().Add(-24 * time.Hour)
	global.DB.Model(&model.Message{}).
		Where("created_at > ? AND user_id IS NOT NULL", last24h).
		Distinct("user_id").
		Count(&activeUsers)
	activeUserGauge.Set(float64(activeUsers))

	// 统计总消息数 (广场用户消息 + 养成系统用户消息)
	var squareMsgCount int64
	global.DB.Model(&model.Message{}).Where("is_ai_reply = ?", false).Count(&squareMsgCount)

	var raisingMsgCount int64
	global.DB.Model(&model.DragonMessage{}).Where("role = ?", "user").Count(&raisingMsgCount)

	messageTotalGauge.Set(float64(squareMsgCount + raisingMsgCount))
}
