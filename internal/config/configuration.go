package config

import (
	"path/filepath"
	"os"
	"encoding/json"
)


const configFileName = ".gatorconfig.json"


type Config struct {
  DBURL string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}


func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configFileName), nil
}


func write(cfg Config) error {
	fileName, err := getConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, data, 0644)
}


func Read() (Config, error) {
	fileName, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}


func (cfg *Config) SetUser(CurrentUserName string) error {
	cfg.CurrentUserName = CurrentUserName	
	return write(*cfg)
}
