package config

import (
	"context"
	"fmt"
	"gopkg.in/natefinch/lumberjack.v2"
	gormlogger "gorm.io/gorm/logger"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level" yaml:"level"`
	Format     string `mapstructure:"format" yaml:"format"`
	FilePath   string `mapstructure:"file_path" yaml:"file_path"`
	MaxSize    int    `mapstructure:"max_size" yaml:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" yaml:"max_backups"`
	MaxAge     int    `mapstructure:"max_age" yaml:"max_age"`
	Compress   bool   `mapstructure:"compress" yaml:"compress"`
}
type GormSlogger struct {
	LogLevel      gormlogger.LogLevel
	SlowThreshold time.Duration
}

// NewGormLogger 创建 GORM 的 slog 适配器
func NewGormLogger(cfg DatabaseConfig) *GormSlogLogger {
	var level gormlogger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		level = gormlogger.Silent
	case "error":
		level = gormlogger.Error
	case "warn":
		level = gormlogger.Warn
	default:
		level = gormlogger.Info
	}

	return &GormSlogLogger{
		LogLevel:      level,
		SlowThreshold: 200 * time.Millisecond,
	}
}

// LogMode 实现 gormlogger.Interface
func (l *GormSlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 实现 gormlogger.Interface
func (l *GormSlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		slog.InfoContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn 实现 gormlogger.Interface
func (l *GormSlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		slog.WarnContext(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error 实现 gormlogger.Interface
func (l *GormSlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		slog.ErrorContext(ctx, fmt.Sprintf(msg, data...))
	}
}
func (l *GormSlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	// 根据执行情况选择日志级别
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error:
		slog.ErrorContext(ctx, "SQL执行错误",
			"sql", sql,
			"rows", rows,
			"elapased_ms", elapsed.Milliseconds(),
			"error", err)
	case elapsed > l.SlowThreshold && l.LogLevel >= gormlogger.Warn:
		slog.WarnContext(ctx, "慢查询",
			"sql", sql,
			"rows", rows,
			"elapsed_ms", elapsed.Milliseconds(),
			"threshold_ms", l.SlowThreshold.Milliseconds())
	case l.LogLevel >= gormlogger.Info:
		slog.InfoContext(ctx, "SQL执行",
			"sql", sql,
			"rows", rows,
			"elapsed_ms", elapsed.Milliseconds())
	}
}

//initSlogWithRotation 初始化带日志切合的slog

func initSlogWithRotation() error {
	//确保日志目录存在
	logDir := filepath.Dir(Config.Log.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 配置 lumberjack 日志切割器
	rotator := &lumberjack.Logger{
		Filename:   Config.Log.FilePath,   // 日志文件路径
		MaxSize:    Config.Log.MaxSize,    // 单个文件最大大小（MB）
		MaxBackups: Config.Log.MaxBackups, // 保留的旧文件数
		MaxAge:     Config.Log.MaxAge,     // 保留天数
		Compress:   Config.Log.Compress,   // 是否压缩
		LocalTime:  true,                  // 使用本地时间
	}

	//设置日志级别
	var level slog.Level
	switch Config.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	// 设置输出格式
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // 添加文件名和行号
	}
	// 创建多输出 Writer（同时输出到文件和终端）
	var writers []io.Writer
	writers = append(writers, rotator) // 输出到文件
	// 如果配置了终端输出（可选，根据环境变量或配置决定）
	if os.Getenv("GRAIN_LOG_CONSOLE") != "false" {
		writers = append(writers, os.Stdout) // 同时输出到终端
	}

	multiWriter := io.MultiWriter(writers...)

	// 创建 handler
	var handler slog.Handler
	switch Config.Log.Format {
	case "json":
		handler = slog.NewJSONHandler(multiWriter, opts)
	default:
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	// 设置全局 logger
	logger = slog.New(handler)
	slog.SetDefault(logger)
	slog.Info("日志系统初始化成功",
		"level", Config.Log.Level,
		"format", Config.Log.Format,
		"file", Config.Log.FilePath,
		"max_size_mb", Config.Log.MaxSize,
		"max_backups", Config.Log.MaxBackups,
		"max_age_days", Config.Log.MaxAge,
		"compress", Config.Log.Compress,
	)
	return nil
}

// GormSlogLogger 实现 GORM 的 logger.Interface
type GormSlogLogger struct {
	LogLevel      gormlogger.LogLevel
	SlowThreshold time.Duration
}
