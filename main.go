package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	HomeAssistantEndpoint string
	HomeAssistantToken    string
}

func main() {
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("homie: ")

	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("failed to get config directory: %v", err)
	}

	configFilePath := filepath.Join(configDir, "homie", "config.json")
	content, err := getConfigContent(configFilePath)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// parse config
	config, err := parseConfig(content)
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	// run
	if config.HomeAssistantEndpoint == "" {
		log.Fatalf("HomeAssistantEndpoint is not set in %s", configFilePath)
	}
	if config.HomeAssistantToken == "" {
		log.Fatalf("HomeAssistantToken is not set in %s", configFilePath)
	}

	run(config)
}

func getConfigContent(configFilePath string) ([]byte, error) {
	// read config
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config := Config{
				HomeAssistantEndpoint: "http://192.168.1.10:8123",
				HomeAssistantToken:    "token",
			}
			content, err = json.Marshal(config)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal config: %w", err)
			}
			err = os.MkdirAll(filepath.Dir(configFilePath), 0o700)
			if err != nil {
				return nil, fmt.Errorf("failed to create config directory: %w", err)
			}
			err = os.WriteFile(configFilePath, content, 0o600)
			if err != nil {
				return nil, fmt.Errorf("failed to write config file: %w", err)
			}

			return nil, fmt.Errorf("config file created, edit it at %v", configFilePath)
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
