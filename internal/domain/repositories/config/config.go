package config

import (
	"github.com/peter7775/alevisualizer/internal/domain/models"

	"github.com/spf13/viper"
)

func Load() (*models.Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config/")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config models.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
