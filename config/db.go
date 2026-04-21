package config

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type DatabaseConfig struct {
	Driver       string `yaml:"driver"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"username"`
	Password     string `yaml:"password"`
	DBName       string `yaml:"dbname"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

var DB *gorm.DB
var Config AppConfig

func (app *Application) initDB() error {
	cfg := mysql.Config{
		DSN: fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
			app.Config.DB.User, app.Config.DB.Password,
			app.Config.DB.Host, app.Config.DB.Port,
			app.Config.DB.DBName, app.Config.DB.Charset),
	}
	var db *gorm.DB
	var err error
	for i := 0; i < 3; i++ {
		db, err = gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
			Logger:                 logger.Default.LogMode(logger.Info),
			SkipDefaultTransaction: true,
		})
		if err != nil {
			break
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	sqlDB.SetMaxIdleConns(app.Config.DB.MaxIdleConns)
	sqlDB.SetMaxOpenConns(app.Config.DB.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	app.DB = db
	return nil
}

func (c *AppConfig) validate() error {
	if c.DB.Host == "" {
		return fmt.Errorf("database host is missing")
	}
	if c.DB.Port == 0 {
		return fmt.Errorf("database port is missing")
	}
	if c.DB.User == "" {
		return fmt.Errorf("database username is missing")
	}
	if c.DB.Password == "" {
		return fmt.Errorf("database password is missing")
	}
	if c.DB.DBName == "" {
		return fmt.Errorf("database name is missing")
	}
	if c.DB.Charset == "" {
		return fmt.Errorf("database charset is missing")
	}
	if c.DB.MaxIdleConns == 0 {
		return fmt.Errorf("database max idle connections is missing")
	}
	if c.DB.MaxOpenConns == 0 {
		return fmt.Errorf("database max open connections is missing")
	}
	if c.DB.Driver == "" {
		return fmt.Errorf("database driver is missing")
	}
	return nil
}
