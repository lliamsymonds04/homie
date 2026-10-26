package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("homie: ")

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

	run(config)
}

func run(config Config) {
	valid, err := checkConnection(config)
	if err != nil {
		log.Fatalf("failed to check connection: %v", err)
	}
	if !valid {
		log.Fatalf("failed to connect to HomeAssistant")
	}
	log.Printf("connected to HomeAssistant")

}

func checkConnection(config Config) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, config.HomeAssistantEndpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+config.HomeAssistantToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("failed to connect to HomeAssistant: %v", resp.StatusCode)
	}

	return true, nil
}
