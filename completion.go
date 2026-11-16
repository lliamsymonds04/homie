package main

import (
	"slices"
	"strings"
)

var commands = []string{"list", "on", "off", "toggle", "state", "light", "clear", "trigger", "completion", "help"}

func Complete(args []string, history []string) []string {
	if len(args) == 0 {
		return commands
	}

	if len(args) > 1 {
		return nil
	}

	switch args[0] {
	case "on", "off", "toggle", "state", "light", "trigger":
		return history
	case "list":
		var domains []string
		for _, entry := range history {
			if domain, _, ok := strings.Cut(entry, "."); ok && !slices.Contains(domains, domain) {
				domains = append(domains, domain)
			}
		}
		return domains
	case "completion":
		return []string{"fish", "bash"}
	default:
		return nil
	}
}

// -f stops fish offering filenames, -k keeps the newest-first history order.
const fishCompletion = `complete -c homie -f -k -a '(homie _complete (commandline -opc)[2..-1])'
`

const bashCompletion = `_homie() {
	local cur="${COMP_WORDS[COMP_CWORD]}"
	local prev="${COMP_WORDS[*]:1:$((COMP_CWORD - 1))}"
	COMPREPLY=($(compgen -W "$(homie _complete $prev)" -- "$cur"))
}
complete -o nosort -F _homie homie
`
