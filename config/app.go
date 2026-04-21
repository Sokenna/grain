package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

type RawString string

func (rs *RawString) UnmarshalYAML(value *yaml.Node) error {
	*rs = RawString(value.Value)
	return nil
}

type AppConfig struct {
	DB *DatabaseConfig `yaml:"database"`
}

type Application struct {
	DB     *gorm.DB
	Config *AppConfig
}

func NewApplication(configPath string) (*Application, error) {
	app := &Application{}
	if err := app.LoadConfig(configPath); err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}
	if err := app.initDB(); err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	return app, nil
}

func (app *Application) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("error unmarshalling config: %w", err)
	}
	if err := config.validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	app.Config = &config
	return nil
}

func (app *Application) Close() error {
	if app.DB != nil {
		sqlDB, err := app.DB.DB()
		if err != nil {
			return fmt.Errorf("error getting database connection: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}
