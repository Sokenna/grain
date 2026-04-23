package auth

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

// HashPassword 加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	if password == "" || hash == "" {
		slog.Warn("密码验证参数为空", "has_password", password != "", "has_hash", hash != "")
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if err != nil {
		// 记录详细错误但不暴露给调用者
		slog.Debug("密码验证失败",
			"error_type", fmt.Sprintf("%T", err),
			"error_msg", err.Error(),
		)
		return false
	}

	return true
}
