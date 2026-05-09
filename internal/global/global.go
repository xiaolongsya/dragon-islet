package global

import (
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	DB     *gorm.DB
	REDIS  *redis.Client
	CONFIG *viper.Viper
	LOG    *zap.Logger
)
