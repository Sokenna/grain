package config_test

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"grain/config"
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfig(t *testing.T) {
	// 准备测试配置
	testConfig := createTestConfigFile(t)
	defer cleanupTestConfig(testConfig)
	// 设置环境变量
	os.Setenv("GRAIN_DB_HOST", "127.0.0.1")
	os.Setenv("GRAIN_DB_PORT", "3306")
	os.Setenv("GRAIN_DB_USERNAME", "root")
	os.Setenv("GRAIN_DB_PASSWORD", "!@#123QWEqwe")
	defer func() {
		os.Unsetenv("GRAIN_DB_HOST")
		os.Unsetenv("GRAIN_DB_PORT")
	}()

	// 注意：这个测试会尝试连接真实数据库，在CI环境中可能需要跳过
	if testing.Short() {
		t.Skip("跳过需要数据库连接的测试")
	}

	// 由于需要真实数据库连接，这里只测试配置加载部分
	err := config.InitConfig()
	assert.NoError(t, err)

}

// 辅助函数：创建测试配置文件
func createTestConfigFile(t *testing.T) string {
	t.Helper()
	configContent := `
server:
  port: 9090

db:
  driver: mysql
  host: testhost.example.com
  port: 3308
  username: testuser
  password: testpass
  dbname: testdb
  charset: utf8mb4
  max_idle_conns: 20
  max_open_conns: 100
  log_level: info
log:
  level: info
  format: json
  file_path: ./test.log
  max_size: 50
  max_backups: 10
  max_age: 3
  compress: false
`
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	viper.SetConfigFile(configPath)
	return configPath
}

// 辅助函数：清理测试配置
func cleanupTestConfig(configPath string) {
	viper.Reset()
	if configPath != "" {
		os.Remove(configPath)
	}
}
