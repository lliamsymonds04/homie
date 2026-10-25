package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
)

type Config struct {
	HomeAssistantEndpoint string
	HomeAssistantToken    string
}

func main() {
	flag.Parse()

	content, configFileErr := getConfigContent()
	if configFileErr != nil {
		panic(configFileErr)
	}

	// parse config
	config, err := parseConfig(content)
	if err != nil {
		panic(err)
	}

	run(config)
}

func getConfigContent() ([]byte, error) {
	var configDir, err = os.UserConfigDir()
	if err != nil {
		log.Fatalf("failed to get config directory: %v", err)
	}

	var configFilePath = configDir + "/homie/config.json"

	// read config
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("config file not found, creating new one at %s", configFilePath)
			config := Config{
				HomeAssistantEndpoint: "http://192.168.1.10:8123",
				HomeAssistantToken:    "token",
			}
			content, err = json.Marshal(config)
			if err != nil {
				log.Fatalf("failed to marshal config: %v", err)
			}
			err = os.MkdirAll(configDir+"/homie", 0o700)
			if err != nil {
				log.Fatalf("failed to create config directory: %v", err)
			}
			err = os.WriteFile(configFilePath, content, 0644)
			if err != nil {
				log.Fatalf("failed to write config file: %v", err)
			}

			return content, fmt.Errorf("config file created, edit it at %s", configFilePath)
		} else if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("config file is not readable: %w", err)
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return content, nil
}

func parseConfig(content []byte) (config Config, err error) {
	// parse config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return
	}

	return
}

func run(config Config) {
	fmt.Println("config:", config)
}
