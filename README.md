# homie

A tiny Home Assistant CLI in Go. List your entities, switch things on and off, check state, and set light colours from a terminal.

No dependencies outside the standard library.

**Note** This is my first ever Go project. I enjoyed the language and I'm happy with the result.

## Requirements

- Go 1.27+ to build
- A Home Assistant instance and a [long-lived access token](https://www.home-assistant.io/docs/authentication/#your-account-profile)

## Install

```sh
go install github.com/lliamsymonds04/homie@latest
```

or from a clone:

```sh
git clone https://github.com/lliamsymonds04/homie.git
cd homie
go build -o ~/.local/bin/homie
```

`homie version` prints the version it was installed at (`dev`-ish pseudo-version for a plain local build).

## Configure

Config lives in your user config directory:

| OS | Path |
| --- | --- |
| Linux | `~/.config/homie/config.json` (`$XDG_CONFIG_HOME` respected) |
| macOS | `~/Library/Application Support/homie/config.json` |
| Windows | `%AppData%\homie\config.json` |

Run `homie` once before setting up: it writes a template and tells you where.

```json
{
  "HomeAssistantEndpoint": "http://192.168.1.10:8123/api",
  "HomeAssistantToken": "token"
}
```

`HomeAssistantEndpoint` is the full API base — it must end in `/api`. Replace the token with a long-lived token from your Home Assistant profile. The file is created with `0600` permissions; keep it that way, it holds a credential.

## Usage

```
homie <command> [args]

list [domain]         list entities, optionally filtered by domain
on|off|toggle <id>    switch an entity
state <id>            print an entity's state
light <id> <colour>   set a light's colour
trigger <id>          run an automation
clear                 clear entity history
completion fish|bash  print a shell completion script
version               print the version
help                  print the above
```

Entity IDs are Home Assistant's `domain.object_id`, e.g. `light.kitchen` or `automation.morning`.

```sh
$ homie list
entity: light.kitchen
entity: switch.fan
entity: automation.morning

$ homie list light        # prefix match on the entity id
entity: light.kitchen

$ homie on light.kitchen
$ homie state light.kitchen
state: on
$ homie light light.kitchen blue
$ homie trigger automation.morning
```

`light` only accepts `light.*` entities and takes any colour name Home Assistant understands (`red`, `blue`, `warm_white`, …). `trigger` only accepts `automation.*`.

## Shell completion

Entity IDs for `on`, `off`, `toggle`, `state`, `light` and `trigger` are completed from your last 50 used entities, newest first. `list` completes domains.

Fish:

```sh
homie completion fish > ~/.config/fish/completions/homie.fish
```

Bash:

```sh
homie completion bash > /etc/bash_completion.d/homie   # or ~/.local/share/bash-completion/completions/homie
```

## Development

```sh
go test ./...
```

## License

MIT — see [LICENSE](LICENSE).
