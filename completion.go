package main

import (
	"slices"
	"strings"
)

var commands = []string{"list", "on", "off", "toggle", "state", "light", "completion"}

func Complete(args []string, history []string) []string {
	if len(args) == 0 {
		return commands
	}

	if len(args) > 1 {
		return nil
	}

	switch args[0] {
	case "on", "off", "toggle", "state", "light":
		return history
	case "list":
		var domains []string
		for _, entry := range history {
			if domain, _, ok := strings.Cut(entry, "."); ok && !slices.Contains(domains, domain) {
				domains = append(domains, domain)
			}
		}
		return domains
	default:
		return nil
	}
}
