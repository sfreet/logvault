package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Syslog struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Protocol string `mapstructure:"protocol"`
	} `mapstructure:"syslog"`
	Redis struct {
		Address  string `mapstructure:"address"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	Web struct {
		Port   int    `mapstructure:"port"`
		Secret string `mapstructure:"secret"`
	} `mapstructure:"web"`
	API struct {
		BearerToken string `mapstructure:"bearer_token"`
	} `mapstructure:"api"`
	ExternalAPI struct {
		Enabled     bool   `mapstructure:"enabled"`
		URL         string `mapstructure:"url"`
		Method      string `mapstructure:"method"`
		BearerToken string `mapstructure:"bearer_token"`
		TriggerTag  string `mapstructure:"trigger_tag"`
	} `mapstructure:"external_api"`
}

// LoadConfig reads configuration from config.yaml
func LoadConfig() (Config, error) {
	var appConfig Config

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("web.port", 8080)
	viper.SetDefault("web.secret", "") // Default empty secret
	viper.SetDefault("syslog.port", 514)
	viper.SetDefault("syslog.host", "0.0.0.0")
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("api.bearer_token", "") // Default empty bearer token
	viper.SetDefault("external_api.enabled", false)
	viper.SetDefault("external_api.url", "")
	viper.SetDefault("external_api.method", "POST")
	viper.SetDefault("external_api.bearer_token", "")
	viper.SetDefault("external_api.trigger_tag", "ALARM")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return appConfig, err // Only return error if it's not a file not found error
		}
	}

	if err := viper.Unmarshal(&appConfig); err != nil {
		return appConfig, err
	}

	return appConfig, nil
}
