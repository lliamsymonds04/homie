package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func GetEntities(config Config) ([]Entity, error) {
	res, err := Get(config, "states")
	if err != nil {
		return nil, err
	}

	var entities []Entity
	if err := json.Unmarshal(res, &entities); err != nil {
		return nil, err
	}

	return entities, nil
}

func SetEntity(config Config, verb string, entityID string) error {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("%q is not domain.object_id", entityID)
	}

	service := verb
	if verb == "on" || verb == "off" {
		service = "turn_" + verb
	}

	if _, err := Set(config, fmt.Sprintf("services/%s/%s", parts[0], service), entityID); err != nil {
		return fmt.Errorf("failed: %v", err)
	}

	return nil
}

func GetState(config Config, entityID string) (string, error) {
	res, err := Get(config, fmt.Sprintf("states/%s", entityID))
	if err != nil {
		return "", err
	}

	var entity Entity
	if err := json.Unmarshal(res, &entity); err != nil {
		return "", err
	}

	return entity.State, nil
}

func Get(config Config, path string) (json.RawMessage, error) {
	full, err := url.JoinPath(config.HomeAssistantEndpoint, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.HomeAssistantToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, fmt.Errorf("token rejected (%s)", resp.Status)
		case http.StatusNotFound:
			return nil, fmt.Errorf("not found %s", full)
		}
		return nil, fmt.Errorf("home assistant returned %s: %s", resp.Status, body)
	}

	return body, nil
}

func Set(config Config, path string, entityId string) (json.RawMessage, error) {
	payload := map[string]string{"entity_id": entityId}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	full, err := url.JoinPath(config.HomeAssistantEndpoint, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, full, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.HomeAssistantToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("home assistant returned %s: %s", resp.Status, out)
	}

	return out, nil
}
