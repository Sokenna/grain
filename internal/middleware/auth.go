package middleware

import (
	"github.com/gin-gonic/gin"
	"grain/config"
	"grain/internal/auth"
	"net/http"
	"strings"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取token
		tokenString := extractToken(c, &config.Config.Auth.JWT)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "未提供认证token",
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := jwtManager.VerifyToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RBACMiddleware RBAC权限中间件
func RBACMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用RBAC，直接放行
		if !config.Config.Auth.RBAC.Enabled {
			c.Next()
			return
		}

		// 获取用户角色
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "未找到用户角色信息",
			})
			c.Abort()
			return
		}

		userRole := role.(string)

		// 检查是否为超级管理员
		for _, superRole := range config.Config.Auth.RBAC.SuperRoles {
			if userRole == superRole {
				c.Next()
				return
			}
		}

		// 检查角色是否在允许列表中
		allowed := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "权限不足",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractToken 提取token
func extractToken(c *gin.Context, jwtConfig *config.JWTConfig) string {
	// 解析 token_lookup 配置
	lookups := strings.Split(jwtConfig.TokenLookup, ",")

	for _, lookup := range lookups {
		parts := strings.SplitN(strings.TrimSpace(lookup), ":", 2)
		if len(parts) != 2 {
			continue
		}

		source := parts[0]
		key := parts[1]

		switch source {
		case "header":
			token := c.GetHeader(key)
			if token != "" {
				// 处理 Bearer token
				if strings.HasPrefix(token, "Bearer ") {
					return strings.TrimPrefix(token, "Bearer ")
				}
				return token
			}
		case "query":
			token := c.Query(key)
			if token != "" {
				return token
			}
		case "cookie":
			token, err := c.Cookie(key)
			if err == nil && token != "" {
				return token
			}
		}
	}

	return ""
}
