package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigContentCreatesFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path, err := getConfigFilePath()
	if err != nil {
		t.Fatalf("get config file path: %v", err)
	}
	if want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "homie", "config.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	if _, err := getConfigContent(path); err == nil {
		t.Fatal("expected created-config error from first call, got nil")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("%s has mode %o, want 600", path, perm)
	}

	content, err := getConfigContent(path)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if _, err := parseConfig(content); err != nil {
		t.Fatalf("created file does not parse: %v", err)
	}
}

func TestGetConfigContentDoesNotClobber(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	path, err := getConfigFilePath()
	if err != nil {
		t.Fatalf("get config file path: %v", err)
	}
	if _, err := getConfigContent(path); err == nil {
		t.Fatal("expected created-config error, got nil")
	}

	if err := os.WriteFile(path, []byte(`{"HomeAssistantToken":"mine"}`), 0o600); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	content, err := getConfigContent(path)
	if err != nil {
		t.Fatalf("call on existing config: %v", err)
	}
	config, err := parseConfig(content)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if config.HomeAssistantToken != "mine" {
		t.Fatalf("token = %q, want %q", config.HomeAssistantToken, "mine")
	}
}

func TestParseConfig(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Config
		wantErr bool
	}{
		{"empty object", `{}`, Config{}, false},
		{"lowercase keys", `{"homeassistanttoken":"lower"}`, Config{HomeAssistantToken: "lower"}, false},
		{"garbage", `nope`, Config{}, true},
		{"wrong type", `[]`, Config{}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseConfig([]byte(tc.in))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseConfig(%q) = %+v, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseConfig(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("parseConfig(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}
