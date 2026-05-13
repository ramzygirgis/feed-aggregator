package config

import (
	"os"
	"fmt"
	"encoding/json"
)

type Config struct {
	DBURL string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}



func Read() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	fileName := fmt.Sprintf("%s/.gatorconfig.json", homeDir)
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


