package main

import (
	"flag"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
)

type Entity struct {
	ID    string `json:"entity_id"`
	State string `json:"state"`
}

func main() {
	flag.Usage = printUsage
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
		case "help":
			printUsage()
			return
		case "version":
			fmt.Println(buildVersion())
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

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if len(args) == 0 {
		log.Fatal(usage)
	}

	if err := run(config, args); err != nil {
		log.Fatalf("%v", err)
	}
}

// buildVersion reads the module version go install stamps in; a plain local
// build has none, so it reports dev.
func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

func loadConfig() (Config, error) {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get config file path: %v", err)
	}

	content, err := getConfigContent(configFilePath)
	if err != nil {
		return Config{}, err
	}

	config, err := parseConfig(content)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config: %v", err)
	}

	if config.HomeAssistantEndpoint == "" {
		return Config{}, fmt.Errorf("HomeAssistantEndpoint is not set in %s", configFilePath)
	}
	if config.HomeAssistantToken == "" {
		return Config{}, fmt.Errorf("HomeAssistantToken is not set in %s", configFilePath)
	}

	return config, nil
}

func run(config Config, args []string) error {
	switch args[0] {
	case "list":
		domain := ""
		if len(args) > 1 {
			domain = args[1]
		}

		return list(config, domain)
	case "on", "off", "toggle":
		if len(args) < 2 {
			return fmt.Errorf("%s no entity specified", args[0])
		}
		if err := SetEntity(config, args[0], args[1]); err != nil {
			return fmt.Errorf("failed to set entity: %v", err)
		}
		RecordHistory(args[1])
	case "state":
		if len(args) < 2 {
			return fmt.Errorf("%s no entity specified", args[0])
		}
		state, err := GetState(config, args[1])
		if err != nil {
			return fmt.Errorf("failed to get state: %v", err)
		}
		RecordHistory(args[1])
		log.Printf("state: %s", state)
	case "light":
		if len(args) < 3 {
			return fmt.Errorf("Exptected 3 arguments, light <entity> <colour>")
		}
		entityId := args[1]
		colour := args[2]
		if err := SetLightColour(config, entityId, colour); err != nil {
			return fmt.Errorf("failed to set light: %v", err)
		}
		RecordHistory(entityId)
	case "trigger":
		if len(args) < 2 {
			return fmt.Errorf("Exptected 2 arguments, trigger <entity>")
		}
		entityId := args[1]
		if err := TriggerAutomation(config, entityId); err != nil {
			return fmt.Errorf("failed to trigger: %v", err)
		}
		RecordHistory(entityId)
	case "clear":
		ClearHistory()
	default:
		return fmt.Errorf("unknown command: %s (see homie help)", args[0])
	}

	return nil
}

func printUsage() {
	fmt.Print(usage)
}

const usage = `usage: homie <command> [args]

commands:
  list [domain]         list entities, optionally filtered by domain
  on|off|toggle <id>    switch an entity
  state <id>            print an entity's state
  light <id> <colour>   set a light's colour
  trigger <id>			run an automation
  clear                 clear entity history
  completion fish|bash  print a shell completion script
  version               print the version
  help                  print this

_complete is used internally by shell completion.
`

func list(config Config, domain string) error {
	entities, err := GetEntities(config)
	if err != nil {
		return fmt.Errorf("failed to get entities: %v", err)
	}

	for _, entity := range entities {
		if domain != "" && !strings.HasPrefix(entity.ID, domain) {
			continue
		}
		log.Printf("entity: %s", entity.ID)
	}

	return nil
}
