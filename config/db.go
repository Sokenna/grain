package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"log/slog"
	"time"
)

type DatabaseConfig struct {
	Driver       string `mapstructure:"driver" yaml:"driver"`
	Host         string `mapstructure:"host" yaml:"host"`
	Port         int    `mapstructure:"port" yaml:"port"`
	User         string `mapstructure:"username" yaml:"username"`
	Password     string `mapstructure:"password" yaml:"password"`
	DBName       string `mapstructure:"dbname" yaml:"dbname"`
	Charset      string `mapstructure:"charset" yaml:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level" yaml:"log_level"` // silent, error, warn, info
}

// buildDSN 构建数据库连接字符串
func buildDSN(cfg DatabaseConfig) string {
	// 从 viper 直接读取
	user := viper.GetString("db.username")
	password := viper.GetString("db.password")
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		user, password,
		cfg.Host, cfg.Port,
		cfg.DBName, cfg.Charset,
	)
}

func initDB() error {
	cfg := Config.DB

	dsn := buildDSN(cfg)

	gormLogger := NewGormLogger(cfg)

	db, err := connectWithRetry(dsn, 3, gormLogger)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}
	if err := configureConnPool(db, cfg); err != nil {
		return fmt.Errorf("配置连接池失败: %w", err)
	}
	DB = db
	return nil
}
func connectWithRetry(dsn string, maxRetries int, gormLogger gormlogger.Interface) (*gorm.DB, error) {
	var lastErr error
	for attempt := 1; attempt < maxRetries; attempt++ {
		slog.Debug("尝试连接数据库", "attempt", attempt, "max_retries", maxRetries)
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger:                 gormLogger,
			SkipDefaultTransaction: true,
		})

		if err == nil {
			if attempt > 1 {
				slog.Info("数据库连接成功", "after_retries", attempt-1)
			}
			return db, nil
		}

		lastErr = err
		slog.Warn("数据库连接失败", "attempt", attempt, "error", err)
		// 最后一次重试失败后不再等待
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * time.Second
			slog.Debug("等待后重试", "wait_seconds", waitTime.Seconds())
			time.Sleep(waitTime)
		}
	}
	return nil, fmt.Errorf("重试%dc次后仍然失败: %w", maxRetries, lastErr)
}

func configureConnPool(db *gorm.DB, cfg DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取底层sql.DB失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 可选：设置空闲连接的最大存活时间
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	slog.Debug("数据库连接池配置",
		"max_idle_conns", cfg.MaxIdleConns,
		"max_open_conns", cfg.MaxOpenConns,
		"conn_max_lifetime", "1h",
		"conn_max_idle_time", "10m",
	)
	return nil
}
func (c *AppConfig) validateMySQLConfig() error {
	if c.DB.Host == "" {
		return errors.New("数据库主机地址不能为空")
	}
	if c.DB.Port <= 0 || c.DB.Port > 65535 {
		return fmt.Errorf("数据库端口无效: %d (范围: 1-65535)", c.DB.Port)
	}
	if c.DB.User == "" {
		return errors.New("数据库用户名不能为空")
	}
	if c.DB.Password == "" {
		return errors.New("数据库密码不能为空")
	}
	if c.DB.DBName == "" {
		return errors.New("数据库名称不能为空")
	}
	if c.DB.Charset == "" {
		return errors.New("数据库字符集不能为空")
	}
	return nil
}

// GetDSN 获取数据库连接字符串
func (c *AppConfig) GetDSN() string {
	return buildDSN(c.DB)
}
