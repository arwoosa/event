package conf

import (
	"flag"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	Mode           string `mapstructure:"mode"`
	Port           int    `mapstructure:"port"`
	Name           string `mapstructure:"name"`
	Version        string `mapstructure:"version"`
	TimeZone       string `mapstructure:"time_zone"`
	*LogConfig     `mapstructure:"log"`
	*MongodbConfig `mapstructure:"mongodb"`
}

// MongodbConfig holds the MongoDB configuration.
type MongodbConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"db"`
}

// LogConfig holds the logger configuration.
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// NewConfig loads the application configuration from a file.
func NewConfig() (*AppConfig, error) {
	var confFile string
	flag.StringVar(&confFile, "c", "./conf/config.yaml", "配置文件")
	flag.Parse()

	v := viper.New()
	v.SetConfigFile(confFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var conf AppConfig
	if err := v.Unmarshal(&conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set timezone
	loc, err := time.LoadLocation(conf.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}
	time.Local = loc

	return &conf, nil
}


