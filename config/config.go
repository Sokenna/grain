package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type AppConfig struct {
	Server ServerConfig   `mapstructure:"server" json:"server"`
	DB     DatabaseConfig `mapstructure:"db" json:"db"`
	Log    LogConfig      `mapstructure:"log" json:"log"`
}
type ServerConfig struct {
	Port int `json:"port"`
}
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

var (
	DB     *gorm.DB
	Config AppConfig
	logger *slog.Logger
)

func InitConfig() error {

	if err := initViper(); err != nil {
		slog.Error("初始化配置失败", "error", err)
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	if err := Config.Validate(); err != nil {
		slog.Error("配置校验失败", "error", err)
		return fmt.Errorf("配置校验失败: %w", err)
	}
	// 初始化日志（必须在配置校验之后）
	if err := initSlogWithRotation(); err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	if err := initDB(); err != nil {
		slog.Error("初始化数据库失败", "error", err)
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	slog.Info("应用初始化成功",
		"server_port", Config.Server.Port,
		"db_host", Config.DB.Host,
	)
	return nil
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

func initViper() error {
	setDefaults()
	// 自动环境变量
	bindEnvVariables()

	setupConfigPaths()
	// 读取配置
	//3. Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}
	// 绑定到结构体
	if err := viper.Unmarshal(&Config); err != nil {
		panic(fmt.Sprintf("配置解析失败: %v", err))
	}
	return nil
}

// bindEnvVariables 绑定环境变量
func bindEnvVariables() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GRAIN")

	// 环境变量映射
	envBindings := map[string]string{
		"db.host":       "GRAIN_DB_HOST",
		"db.port":       "GRAIN_DB_PORT",
		"db.username":   "GRAIN_DB_USER",
		"db.password":   "GRAIN_DB_PASSWORD",
		"db.dbname":     "GRAIN_DB_NAME",
		"log.level":     "GRAIN_LOG_LEVEL",
		"log.file_path": "GRAIN_LOG_FILE",
	}

	for key, env := range envBindings {
		_ = viper.BindEnv(key, env)
	}
}

// setupConfigPaths 设置配置文件搜索路径
func setupConfigPaths() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.grain")
	viper.AddConfigPath("/etc/grain") // 增加系统级配置路径
}

// buildDSN 构建数据库连接字符串
func buildDSN(cfg DatabaseConfig) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password,
		cfg.Host, cfg.Port,
		cfg.DBName, cfg.Charset,
	)
}
func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("db.driver", "mysql")
	viper.SetDefault("db.host", "127.0.0.1")
	viper.SetDefault("db.port", 3306)
	viper.SetDefault("db.charset", "utf8mb4")
	viper.SetDefault("db.max_idle_conns", 10)
	viper.SetDefault("db.max_open_conns", 50)
	viper.SetDefault("db.log_level", "info")

	// 日志默认值
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.file_path", "./logs/grain.log")
	viper.SetDefault("log.max_size", 100)   // 100MB
	viper.SetDefault("log.max_backups", 30) // 保留30个备份
	viper.SetDefault("log.max_age", 7)      // 保留7天
	viper.SetDefault("log.compress", true)  // 压缩旧文件
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
func (c *AppConfig) Validate() error {
	//服务器配置校验
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d (范围: 1-65535)", c.Server.Port)
	}
	//数据库配置校验
	if c.DB.Driver == "" {
		return errors.New("数据库驱动不能为空")
	}
	//支持的数据驱动类型校验
	supportedDrivers := map[string]bool{"mysql": true, "postgres": true, "sqlite": true}
	if !supportedDrivers[c.DB.Driver] {
		return fmt.Errorf("不支持的数据库驱动: %s", c.DB.Driver)
	}
	if c.DB.Driver == "mysql" {
		if err := c.validateMySQLConfig(); err != nil {
			return fmt.Errorf("最大空闲连接数(%d)不能大于最大打开连接数(%d)",
				c.DB.MaxIdleConns, c.DB.MaxOpenConns)
		}
	}

	if c.DB.MaxIdleConns <= 0 {
		return fmt.Errorf("最大空闲连接数无效: %d (必须 >0)", c.DB.MaxIdleConns)
	}
	if c.DB.MaxOpenConns <= 0 {
		return fmt.Errorf("最大打开连接数无效: %d (必须 >0)", c.DB.MaxOpenConns)
	}
	if c.DB.MaxIdleConns > c.DB.MaxOpenConns {
		return fmt.Errorf("最大空闲连接数(%d)不能大于最大打开连接数(%d)",
			c.DB.MaxIdleConns, c.DB.MaxOpenConns)
	}

	// 日志配置校验
	if c.Log.FilePath == "" {
		return errors.New("日志文件路径不能为空")
	}
	if c.Log.MaxSize <= 0 {
		return fmt.Errorf("日志文件最大大小无效: %d (必须 >0)", c.Log.MaxSize)
	}
	if c.Log.MaxBackups < 0 {
		return fmt.Errorf("日志备份数无效: %d (必须 >=0)", c.Log.MaxBackups)
	}
	if c.Log.MaxAge < 0 {
		return fmt.Errorf("日志保留天数无效: %d (必须 >=0)", c.Log.MaxAge)
	}
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

// IsProd 判断是否为生产环境
func (c *AppConfig) IsProd() bool {
	return viper.GetString("env") == "production"
}
