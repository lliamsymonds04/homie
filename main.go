package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Entity struct {
	ID    string `json:"entity_id"`
	State string `json:"state"`
}

func main() {
	flag.Parse()

	log.SetFlags(0)

	configFilePath, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("failed to get config file path: %v", err)
	}

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

	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("no command specified")
	}

	run(config, args)
}

func run(config Config, args []string) {
	switch args[0] {
	case "list":
		domain := ""
		if len(args) > 1 {
			domain = args[1]
		}

		list(config, domain)
	case "on", "off", "toggle":
		if len(args) < 2 {
			log.Fatalf("%s no entity specified", args[0])
		}
		setEntity(config, args[0], args[1])
	default:
		log.Fatalf("unknown command: %s", args[0])
		os.Exit(2)
	}
}

func list(config Config, domain string) {
	entities, err := getEntities(config)
	if err != nil {
		log.Fatalf("failed to get entities: %v", err)
	}

	for _, entity := range entities {
		if domain != "" && !strings.HasPrefix(entity.ID, domain) {
			continue
		}
		log.Printf("entity: %s", entity.ID)
	}
}

func getEntities(config Config) ([]Entity, error) {
	res, err := get(config, "states")
	if err != nil {
		return nil, err
	}

	var entities []Entity
	if err := json.Unmarshal(res, &entities); err != nil {
		log.Fatalf("failed to unmarshal entities: %v", err)
	}

	return entities, nil
}

func setEntity(config Config, verb, entityID string) {
	parts := strings.SplitN(entityID, ".", 2)
	if len(parts) != 2 {
		log.Fatalf("%q is not domain.object_id", entityID)
	}

	service := verb
	if verb == "on" || verb == "off" {
		service = "turn_" + verb
	}

	if _, err := set(config, fmt.Sprintf("services/%s/%s", parts[0], service), entityID); err != nil {
		log.Fatalf("failed: %v", err)
	}
}

func get(config Config, path string) (json.RawMessage, error) {
	full, err := url.JoinPath(config.HomeAssistantEndpoint, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, full, nil)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return nil, fmt.Errorf("token rejected (%s)", resp.Status)
		case http.StatusNotFound:
			return nil, fmt.Errorf("no API at %s", full)
		}
		return nil, fmt.Errorf("home assistant returned %s: %s", resp.Status, body)
	}

	return body, nil
}

func set(config Config, path string, entityId string) (json.RawMessage, error) {
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

	resp, err := http.DefaultClient.Do(req)
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
