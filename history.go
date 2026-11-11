package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const HistoryLimit = 50

func GetHistoryFilePath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "history"), nil
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
