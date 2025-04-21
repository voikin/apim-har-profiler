package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/voikin/apim-har-profiler/pkg/logger"
)

type Config struct {
	Logger *logger.Config `mapstructure:"logger"`
	Server *Server        `mapstructure:"server"`
}

type Server struct {
	GRPC *GRPC `mapstructure:"grpc"`
	HTTP *HTTP `mapstructure:"http"`
}

type GRPC struct {
	Host              string `mapstructure:"host"`
	Port              int    `mapstructure:"port"`
	MaxConnAgeSeconds int    `mapstructure:"max_conn_age_seconds"`
}

type HTTP struct {
	Host                  string `mapstructure:"host"`
	Port                  int    `mapstructure:"port"`
	ReadTimeoutSecs       int    `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSecs      int    `mapstructure:"write_timeout_seconds"`
	ReadHeaderTimeoutSecs int    `mapstructure:"read_header_timeout_seconds"`
}

func (c GRPC) MaxConnectionAge() time.Duration {
	return time.Duration(c.MaxConnAgeSeconds) * time.Second
}

func (c HTTP) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSecs) * time.Second
}

func (c HTTP) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutSecs) * time.Second
}

func (c HTTP) ReadHeaderTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSecs) * time.Second
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err := viper.ReadInConfig()
	if err != nil {
		return &Config{}, fmt.Errorf("viper.ReadInConfig: %w", err)
	}

	config := &Config{}

	err = viper.Unmarshal(config)
	if err != nil {
		return &Config{}, fmt.Errorf("viper.Unmarshal: %w", err)
	}

	return config, nil
}
