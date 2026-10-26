package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
	"strings"
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
		list(config)
	case "lights":
		lights(config)
	default:
		log.Fatalf("unknown command: %s", args[0])
		os.Exit(2)
	}
}

func list(config Config) {
	entities, err := getEntities(config)
	if err != nil {
		log.Fatalf("failed to get entities: %v", err)
	}

	for _, entity := range entities {
		log.Printf("entity: %s", entity.ID)
	}
}

func lights(config Config) {
	entities, err := getEntities(config)
	if err != nil {
		log.Fatalf("failed to get entities: %v", err)
	}

	for _, entity := range entities {
		if !strings.Contains(entity.ID, "light") {
			continue
		}
		if strings.Contains(entity.ID, "automation") {
			continue
		}
		if strings.Contains(entity.ID,"select") {
			continue
		}
		log.Printf("entity: %s, state: %s", entity.ID, entity.State)
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
