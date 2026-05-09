package initialize

import (
	"dragon-islet/internal/global"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func InitConfig() {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")

	// 读取环境变量
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// 如果配置文件不存在，我们尝试从环境变量读取关键信息
		fmt.Printf("Warning: config.yaml not found, using env variables: %v\n", err)
	}
	
	// 允许通过环境变量覆盖
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		v.Set("deepseek.api_key", os.Getenv("DEEPSEEK_API_KEY"))
	}
	if os.Getenv("DB_DSN") != "" {
		v.Set("db.dsn", os.Getenv("DB_DSN"))
	}
	if os.Getenv("REDIS_ADDR") != "" {
		v.Set("redis.addr", os.Getenv("REDIS_ADDR"))
	}

	if os.Getenv("REDIS_PASSWORD") != "" {
		v.Set("redis.password", os.Getenv("REDIS_PASSWORD"))
	}

	global.CONFIG = v
}
