package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Config struct {
	HomeAssistantEndpoint string
	HomeAssistantToken    string
}

const HistoryLimit = 50

func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %v", err)
	}

	return filepath.Join(configDir, "homie"), nil
}

func getConfigFilePath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

func GetHistoryFilePath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "history"), nil
}

func getConfigContent(configFilePath string) ([]byte, error) {
	// read config
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			config := Config{
				HomeAssistantEndpoint: "http://192.168.1.10:8123/api",
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

func LoadHistory() ([]string, error) {
	historyFilePath, err := GetHistoryFilePath()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(historyFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		} else if errors.Is(err, os.ErrPermission) {
			return nil, fmt.Errorf("history file is not readable: %w", err)
		} else {
			return nil, fmt.Errorf("read history file: %w", err)
		}
	}

	return strings.Fields(string(content)), nil
}

func RecordHistory(entry string) error {
	historyFilePath, err := GetHistoryFilePath()
	if err != nil {
		return err
	}

	history, err := LoadHistory()
	if err != nil {
		return err
	}

	history = slices.DeleteFunc(history, func(e string) bool { return e == entry })
	history = append([]string{entry}, history...)
	if len(history) > HistoryLimit {
		history = history[:HistoryLimit]
	}

	content := []byte(strings.Join(history, "\n"))
	os.MkdirAll(filepath.Dir(historyFilePath), 0o700)
	err = os.WriteFile(historyFilePath, content, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("history file is not writable: %w", err)
		} else {
			return fmt.Errorf("write history file: %w", err)
		}
	}

	return nil
}
