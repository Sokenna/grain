package config

import "time"

// ServerConfig 服务配置
type ServerConfig struct {
	Port         int           `mapstructure:"port" json:"port"`
	Host         string        `mapstructure:"host" json:"host"`
	Mode         string        `mapstructure:"mode" json:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" json:"idle_timeout"`

	MaxHeaderBytes int `mapstructure:"max_header_bytes", json:"max_header_bytes"`
	//限流配置
	RateLimit RateLimitConfig `mapstructure:"rate_limit" json:"rate_limit"`
	//TLS/HTTPS配置
	TLS struct {
		Enabled  bool   `mapstructure:"enabled" json:"enabled"`
		CertFile string `mapstructure:"cert_file" json:"cert_file"`
		KeyFile  string `mapstructure:"key_file" json:"key_file"`
	} `mapstructure:"tls" json:"tls"`

	//优雅关闭配置
	GracefulShutdown struct {
		Timeout time.Duration `mapstructure:"timeout" json:"timeout"` //关闭超时时间
	} `mapstructure:"graceful_shutdown" json:"graceful_shutdown"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled      bool            `mapstructure:"enabled" json:"enabled"`
	DefaultLimit int             `mapstructure:"default_limit" json:"default_limit"` // 默认每秒请求数
	DefaultBurst int             `mapstructure:"default_burst" json:"default_burst"` // 默认突发容量
	Rules        []RateLimitRule `mapstructure:"rules" json:"rules"`                 // 路径规则
	SkipPaths    []PathSkipRule  `mapstructure:"skip_paths" json:"skip_paths"`       // 跳过限流的路径
}

// RateLimitRule 限流规则
type RateLimitRule struct {
	Paths   []string `mapstructure:"paths" json:"paths"`       // 匹配的路径列表（支持前缀匹配）
	Methods []string `mapstructure:"methods" json:"methods"`   // 匹配的方法列表（为空则匹配所有方法）
	Limit   int      `mapstructure:"limit" json:"limit"`       // 每秒请求数
	Burst   int      `mapstructure:"burst" json:"burst"`       // 突发容量
	KeyType string   `mapstructure:"key_type" json:"key_type"` // 限流键类型: ip, user_id, api_key
}

// PathSkipRule 路径跳过规则
type PathSkipRule struct {
	Path    string   `mapstructure:"path" json:"path"`       // 路径（支持前缀匹配）
	Methods []string `mapstructure:"methods" json:"methods"` // 跳过的方法（为空则跳过所有方法）
	Exact   bool     `mapstructure:"exact" json:"exact"`     // 是否精确匹配（false为前缀匹配）
}
