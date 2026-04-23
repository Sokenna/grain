package config

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"log/slog"
)

type AppConfig struct {
	Server ServerConfig   `mapstructure:"server" json:"server"`
	DB     DatabaseConfig `mapstructure:"db" json:"db"`
	Log    LogConfig      `mapstructure:"log" json:"log"`
	Auth   AuthConfig     `mapstructure:"auth" json:"auth"`
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

func setDefaults() {
	// 服务器默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "120s")
	viper.SetDefault("server.max_header_bytes", 1048576)

	//限流默认值
	viper.SetDefault("server.rate_limit.enabled", true)
	viper.SetDefault("server.rate_limit.requests", 100)
	viper.SetDefault("server.rate_limit.burst", 200)

	//TLS 默认值
	viper.SetDefault("server.tls.enabled", false)

	//优雅关闭默认值
	viper.SetDefault("server.graceful_shutdown.timeout", "30s")

	// 数据库默认值
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

	// JWT 默认值
	viper.SetDefault("auth.jwt.secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt.expire_time", "24h")
	viper.SetDefault("auth.jwt.refresh_time", "168h") // 7天
	viper.SetDefault("auth.jwt.issuer", "grain-api")
	viper.SetDefault("auth.jwt.token_lookup", "header:Authorization,query:token,cookie:token")

	// RBAC 默认值
	viper.SetDefault("auth.rbac.enabled", true)
	viper.SetDefault("auth.rbac.super_roles", []string{"admin", "super_admin"})
	viper.SetDefault("auth.rbac.default_role", "guest")
}

func (c *AppConfig) Validate() error {
	//服务器配置校验
	if c.Server.Port < 0 || c.Server.Port > 65535 {
		return fmt.Errorf("服务器端口无效: %d (范围: 1-65535)", c.Server.Port)
	}
	if c.Server.Host == "" {
		return errors.New("服务器主机地址不能为空")
	}
	validModes := map[string]bool{"release": true, "debug": true, "test": true}
	if !validModes[c.Server.Mode] {
		return fmt.Errorf("无效的运行模式:%s (支持: debug, test, release)", c.Server.Mode)
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("读取超时时间无效: %v", c.Server.ReadTimeout)
	}

	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("写入超时时间无效: %v", c.Server.WriteTimeout)
	}
	// TLS 配置校验
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return errors.New("TLS 证书文件路径不能为空")
		}
		if c.Server.TLS.KeyFile == "" {
			return errors.New("TLS 私钥文件路径不能为空")
		}
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
			return err
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

// IsProd 判断是否为生产环境
func (c *AppConfig) IsProd() bool {
	return viper.GetString("env") == "production"
}
