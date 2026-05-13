package config

import (
	"os"
	"fmt"
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

	fileName := fmt.Sprintf("%s%s", homeDir, configFileName)
	return fileName, nil
}


func write(c Config) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	//TODO: complete implementation

}

func Read() (Config, error) {
	fileName, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	file, err := os.Open(fileName)	
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	data, err := os.ReadRead(data)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}
	
	return config, nil
}


func (c *Config) SetUser(CurrentUsername: string) {
	c.CurrentUserName = CurrentUsername

}
