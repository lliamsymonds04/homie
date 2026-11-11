package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

type Entity struct {
	ID    string `json:"entity_id"`
	State string `json:"state"`
}

func main() {
	flag.Parse()

	log.SetFlags(0)

	args := flag.Args()

	if len(args) > 0 {
		switch args[0] {
		case "_complete":
			history, _ := LoadHistory()
			for _, candidate := range Complete(args[1:], history) {
				fmt.Println(candidate)
			}
			return
		case "completion":
			if len(args) < 2 {
				log.Fatalf("usage: homie completion fish|bash")
			}
			switch args[1] {
			case "fish":
				fmt.Print(fishCompletion)
			case "bash":
				fmt.Print(bashCompletion)
			default:
				log.Fatalf("no completion for %q, expected fish or bash", args[1])
			}
			return
		}
	}

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
		err := SetEntity(config, args[0], args[1])
		if err != nil {
			log.Fatalf("failed to set entity: %v", err)
		}
		RecordHistory(args[1])
	case "state":
		if len(args) < 2 {
			log.Fatalf("%s no entity specified", args[0])
		}
		state, err := GetState(config, args[1])
		if err != nil {
			log.Fatalf("failed to get state: %v", err)
		}
		RecordHistory(args[1])
		log.Printf("state: %s", state)
	case "light":
		if len(args) < 3 {
			log.Fatalf("Exptected 3 arguments, light <entity> <colour>")
		}
		entity_id := args[1]
		colour := args[2]
		err := SetLightColour(config, entity_id, colour)
		if err != nil {
			log.Fatalf("failed to set light: %v", err)
		}
		RecordHistory(entity_id)
	case "clear":
		ClearHistory()

	default:
		log.Fatalf("unknown command: %s", args[0])
	}
}

func list(config Config, domain string) {
	entities, err := GetEntities(config)
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
