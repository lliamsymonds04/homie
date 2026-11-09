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

	if _, err := Set(config, fmt.Sprintf("services/%s/%s", parts[0], service), entityID, nil); err != nil {
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

func Set(config Config, path string, entityId string, payload map[string]string) (json.RawMessage, error) {
	if payload == nil {
		payload = map[string]string{}
	}
	payload["entity_id"] = entityId

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

func SetLightColour(config Config, entityID string, colour string) error {
	payload := map[string]string{"color_name": colour}
	domain := "light"
	if parts := strings.Split(entityID, "."); len(parts) == 2 {
		v := parts[0]
		if v != domain {
			return fmt.Errorf("Entity %s is not from the %s domain", entityID, domain)
		}
	} else {
		return fmt.Errorf("Entity has no domain, needs to be from %s", domain)
	}

	if _, err := Set(config, fmt.Sprintf("services/light/%s", "turn_on"), entityID, payload); err != nil {
		return fmt.Errorf("failed: %v", err)
	}
	return nil
}
