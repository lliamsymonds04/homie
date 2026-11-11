package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestLoadHistory(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		dir     bool
		want    []string
		wantErr bool
	}{
		{"empty object", "", false, []string{}, false},
		{"valid history", "one two three", false, []string{"one", "two", "three"}, false},
		{"path is directory", "", true, []string{}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			path, err := GetHistoryFilePath()
			if err != nil {
				t.Fatalf("GetHistoryPath: %v", err)
			}
			if tc.dir {
				os.MkdirAll(path, 0o700)
			} else {
				os.MkdirAll(filepath.Dir(path), 0o700)
				os.WriteFile(path, []byte(tc.in), 0o600)
			}

			history, err := LoadHistory()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("LoadHistory(%q) = %+v, want error", tc.in, history)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadHistory(%q): %v", tc.in, err)
			}
			if !reflect.DeepEqual(history, tc.want) {
				t.Errorf("LoadHistory(%q) = %+v, want %+v", tc.in, history, tc.want)
			}

		})

	}
}

func TestRecordHistory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := RecordHistory("one")
	if err != nil {
		t.Fatal(err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if !reflect.DeepEqual(history, []string{"one"}) {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{"one"})
	}

	err = RecordHistory("two")
	if err != nil {
		t.Fatal(err)
	}

	history, err = LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if !reflect.DeepEqual(history, []string{"two", "one"}) {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{"two", "one"})
	}

	// Test duplicates aren't added
	err = RecordHistory("two")
	if err != nil {
		t.Fatal(err)
	}

	history, err = LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if !reflect.DeepEqual(history, []string{"two", "one"}) {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{"two", "one"})
	}
}

func TestRecordHistoryLimit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	for i := range HistoryLimit + 1 {
		if err := RecordHistory(fmt.Sprintf("light.e%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != HistoryLimit {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{"light.e" + fmt.Sprint(HistoryLimit)})
	}

	// check the first entry is the last entry we added
	if want := fmt.Sprintf("light.e%d", HistoryLimit); history[0] != want {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{want})
	}

	// check the very first entry is the one that was dropped
	for _, e := range history {
		if e == "light.e0" {
			t.Error("history[0] entry light.e0 should been dropped")
		}
	}
}
