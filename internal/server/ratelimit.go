package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"grain/config"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiterManager 限流管理器
type RateLimiterManager struct {
	config         *config.RateLimitConfig
	defaultLimiter *limiter.Limiter
	ruleLimiters   map[string]*limiter.Limiter // key: rule index + path
	mu             sync.RWMutex
}

// NewRateLimiterManager 创建限流管理器
func NewRateLimiterManager(cfg *config.RateLimitConfig) *RateLimiterManager {
	manager := &RateLimiterManager{
		config:       cfg,
		ruleLimiters: make(map[string]*limiter.Limiter),
	}

	// 初始化默认限流器
	if cfg.Enabled && cfg.DefaultLimit > 0 {
		rate := limiter.Rate{
			Period: 1 * time.Second,
			Limit:  int64(cfg.DefaultLimit),
		}
		burst := cfg.DefaultBurst
		if burst <= 0 {
			burst = cfg.DefaultLimit
		}
		manager.defaultLimiter = limiter.New(memory.NewStore(), rate)
	}

	// 初始化规则限流器
	for i, rule := range cfg.Rules {
		rate := limiter.Rate{
			Period: 1 * time.Second,
			Limit:  int64(rule.Limit),
		}
		burst := rule.Burst
		if burst <= 0 {
			burst = rule.Limit
		}
		limiterInstance := limiter.New(memory.NewStore(), rate)

		// 为每个规则创建多个路径映射
		for _, path := range rule.Paths {
			key := fmt.Sprintf("%d:%s", i, path)
			manager.ruleLimiters[key] = limiterInstance
		}
	}

	slog.Info("限流管理器初始化完成",
		"enabled", cfg.Enabled,
		"default_limit", cfg.DefaultLimit,
		"rules_count", len(cfg.Rules),
		"skip_paths_count", len(cfg.SkipPaths),
	)

	return manager
}

// GetLimiter 获取匹配的限流器
func (m *RateLimiterManager) GetLimiter(path, method string) *limiter.Limiter {
	if !m.config.Enabled {
		return nil
	}

	if m.shouldSkip(path, method) {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var matchedLimiter *limiter.Limiter
	maxPathLen := 0

	for key, limiterInstance := range m.ruleLimiters {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		rulePath := parts[1]

		if strings.HasPrefix(path, rulePath) {
			rule := m.getRuleByPath(rulePath)
			if rule != nil && m.matchMethod(method, rule.Methods) {
				if len(rulePath) > maxPathLen {
					maxPathLen = len(rulePath)
					matchedLimiter = limiterInstance
				}
			}
		}
	}

	if matchedLimiter != nil {
		return matchedLimiter
	}

	return m.defaultLimiter
}

// shouldSkip 检查是否应该跳过限流
func (m *RateLimiterManager) shouldSkip(path, method string) bool {
	for _, rule := range m.config.SkipPaths {
		matched := false
		if rule.Exact {
			matched = path == rule.Path
		} else {
			matched = strings.HasPrefix(path, rule.Path)
		}

		if matched && m.matchMethod(method, rule.Methods) {
			slog.Debug("跳过限流", "path", path, "method", method)
			return true
		}
	}
	return false
}

// getRuleByPath 根据路径获取规则
func (m *RateLimiterManager) getRuleByPath(path string) *config.RateLimitRule {
	for i, rule := range m.config.Rules {
		for _, rulePath := range rule.Paths {
			if strings.HasPrefix(path, rulePath) {
				return &m.config.Rules[i]
			}
		}
	}
	return nil
}

// matchMethod 检查方法是否匹配
func (m *RateLimiterManager) matchMethod(method string, methods []string) bool {
	if len(methods) == 0 {
		return true
	}
	methodUpper := strings.ToUpper(method)
	for _, m := range methods {
		if strings.ToUpper(m) == methodUpper {
			return true
		}
	}
	return false
}

// GetKey 获取限流键
func (m *RateLimiterManager) GetKey(c *gin.Context, rule *config.RateLimitRule) string {
	keyType := "ip"
	if rule != nil && rule.KeyType != "" {
		keyType = rule.KeyType
	}

	switch keyType {
	case "user_id":
		if userID, exists := c.Get("user_id"); exists {
			return fmt.Sprintf("user:%v", userID)
		}
		return m.getClientIP(c)
	case "api_key":
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			return fmt.Sprintf("api_key:%s", apiKey)
		}
		return m.getClientIP(c)
	default:
		return m.getClientIP(c)
	}
}

// getClientIP 获取客户端IP（支持代理场景）
func (m *RateLimiterManager) getClientIP(c *gin.Context) string {
	// 支持 X-Forwarded-For
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 支持 X-Real-IP
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return xri
	}

	return c.ClientIP()
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(manager *RateLimiterManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		limiterInstance := manager.GetLimiter(path, method)
		if limiterInstance == nil {
			c.Next()
			return
		}

		rule := manager.getRuleByPath(path)
		key := manager.GetKey(c, rule)

		ctx := c.Request.Context()
		limiterCtx, err := limiterInstance.Get(ctx, key)
		if err != nil {
			slog.Error("限流检查失败", "error", err, "key", key)
			c.Next()
			return
		}

		// 设置限流头信息
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiterCtx.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiterCtx.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limiterCtx.Reset))

		if limiterCtx.Reached {
			slog.Warn("请求被限流",
				"path", path,
				"method", method,
				"key", key,
				"limit", limiterCtx.Limit,
			)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "请求过于频繁，请稍后再试",
				"retry_after": limiterCtx.Reset,
			})
			return
		}

		c.Next()
	}
}
