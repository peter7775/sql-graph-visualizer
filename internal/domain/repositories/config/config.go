/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"os"
	"path/filepath"
	"mysql-graph-visualizer/internal/domain/models"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Load() (*models.Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(findProjectRoot() + "/config/")

	logrus.Infof("Načítám konfiguraci z YAML souboru...")
	if err := viper.ReadInConfig(); err != nil {
		logrus.Errorf("Chyba při načítání konfigurace: %v", err)
		return nil, err
	}

	logrus.Infof("Konfigurace načtena: %+v", viper.AllSettings())

	var config models.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Nelze získat pracovní adresář: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			logrus.Fatalf("Nelze najít kořenový adresář projektu")
			return ""
		}
		wd = parent
	}
}