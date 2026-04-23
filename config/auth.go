package config

import "time"

// AuthConfig 认证配置
type AuthConfig struct {
	JWT  JWTConfig  `mapstructure:"jwt" json:"jwt"`
	RBAC RBACConfig `mapstructure:"rbac" json:"rbac"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string        `mapstructure:"secret" json:"secret"`             //JWT密钥
	ExpireTime  time.Duration `mapstructure:"expire_time" json:"expire_time"`   //过期时间
	RefreshTime time.Duration `mapstructure:"refresh_time" json:"refresh_time"` //刷新时间
	Issuer      string        `mapstructure:"issuer" json:"issuer"`             //签发者
	TokenLookup string        `mapstructure:"token_lookup" json:"token_lookup"` // token获取方式: header:Authorization, query:token, cookie:token
}

// RBACConfig RBAC配置
type RBACConfig struct {
	Enabled     bool     `mapstructure:"enabled" json:"enabled"`
	SuperRoles  []string `mapstructure:"super_roles" json:"super_roles"`   // 超级管理员角色
	DefaultRole string   `mapstructure:"default_role" json:"default_role"` // 默认角色
}
