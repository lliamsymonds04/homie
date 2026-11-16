package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The binary is built once so one smoke test can check main's fatal/exit
// wiring; everything else tests run and loadConfig in-process.
var homieBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "homie-test-bin")
	if err != nil {
		panic(err)
	}
	homieBin = filepath.Join(dir, "homie")
	if out, err := exec.Command("go", "build", "-o", homieBin, ".").CombinedOutput(); err != nil {
		panic("go build: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type cmdResult struct {
	stdout, stderr string
	exit           int
}

func runHomie(t *testing.T, configDir string, args ...string) cmdResult {
	t.Helper()

	cmd := exec.Command(homieBin, args...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	res := cmdResult{}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run homie %v: %v", args, err)
		}
		res.exit = exitErr.ExitCode()
	}
	res.stdout, res.stderr = stdout.String(), stderr.String()
	return res
}

func writeConfig(t *testing.T, configDir, content string) {
	t.Helper()

	dir := filepath.Join(configDir, "homie")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// captureLog redirects the log package for the rest of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	})
	return &buf
}

func TestBinary(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		seed    bool
		wantOut string // stdout substring
		wantErr string // stderr substring
		exit    int
	}{
		{"help", []string{"help"}, false, "usage: homie", "", 0},
		{"no args", nil, true, "", "usage: homie", 1},
		{"unknown command", []string{"bogus"}, true, "", "unknown command: bogus", 1},
		{"completion fish", []string{"completion", "fish"}, false, fishCompletion, "", 0},
		{"_complete no args", []string{"_complete"}, false, "list\n", "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.seed {
				writeConfig(t, dir, `{"HomeAssistantEndpoint":"http://127.0.0.1:1/api","HomeAssistantToken":"x"}`)
			}

			res := runHomie(t, dir, tc.args...)
			if res.exit != tc.exit {
				t.Errorf("exit = %d, want %d (stderr %q)", res.exit, tc.exit, res.stderr)
			}
			if tc.wantOut != "" && !strings.Contains(res.stdout, tc.wantOut) {
				t.Errorf("stdout = %q, want %q", res.stdout, tc.wantOut)
			}
			if tc.wantErr != "" && !strings.Contains(res.stderr, tc.wantErr) {
				t.Errorf("stderr = %q, want %q", res.stderr, tc.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  string
		want string // error substring
	}{
		{"no endpoint", `{"HomeAssistantToken":"x"}`, "HomeAssistantEndpoint is not set"},
		{"no token", `{"HomeAssistantEndpoint":"http://127.0.0.1:1/api"}`, "HomeAssistantToken is not set"},
		{"invalid json", `nope`, "invalid config"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			writeConfig(t, dir, tc.cfg)

			_, err := loadConfig()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("loadConfig = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestLoadConfigCreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	_, err := loadConfig()
	if err == nil || !strings.Contains(err.Error(), "config file created, edit it at") {
		t.Fatalf("loadConfig = %v, want config-created message", err)
	}

	info, err := os.Stat(filepath.Join(dir, "homie", "config.json"))
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config mode = %o, want 600", perm)
	}
}

func TestRunErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "boom")
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}

	cases := []struct {
		name string
		args []string
		want string // error substring
	}{
		{"unknown command", []string{"bogus"}, "unknown command: bogus"},
		{"on without entity", []string{"on"}, "on no entity specified"},
		{"off without entity", []string{"off"}, "off no entity specified"},
		{"toggle without entity", []string{"toggle"}, "toggle no entity specified"},
		{"state without entity", []string{"state"}, "state no entity specified"},
		{"light without colour", []string{"light", "light.a"}, "Exptected 3 arguments"},
		{"trigger without entity", []string{"trigger"}, "Exptected 2 arguments"},
		{"on bad entity", []string{"on", "nope"}, "failed to set entity"},
		{"light wrong domain", []string{"light", "switch.a", "red"}, "failed to set light"},
		{"trigger wrong domain", []string{"trigger", "light.a"}, "failed to trigger"},
		{"state api error", []string{"state", "light.a"}, "failed to get state"},
		{"list api error", []string{"list"}, "failed to get entities"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())

			err := run(cfg, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run(%v) = %v, want error containing %q", tc.args, err, tc.want)
			}
		})
	}
}

func TestRunCommands(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/states" {
			io.WriteString(w, `[{"entity_id":"light.a"},{"entity_id":"switch.b"}]`)
			return
		}
		io.WriteString(w, `{"entity_id":"light.a","state":"on"}`)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}

	cases := []struct {
		name    string
		args    []string
		wantLog string // log output substring, empty means no output
	}{
		{"state", []string{"state", "light.a"}, "state: on"},
		{"on", []string{"on", "light.a"}, ""},
		{"off", []string{"off", "light.a"}, ""},
		{"toggle", []string{"toggle", "light.a"}, ""},
		{"light colour", []string{"light", "light.a", "red"}, ""},
		{"trigger", []string{"trigger", "automation.morning"}, ""},
		{"list all", []string{"list"}, "entity: light.a"},
		{"list domain", []string{"list", "switch"}, "entity: switch.b"},
		{"clear", []string{"clear"}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			out := captureLog(t)

			if err := run(cfg, tc.args); err != nil {
				t.Fatalf("run(%v) = %v, want nil", tc.args, err)
			}
			if tc.wantLog == "" {
				if out.Len() != 0 {
					t.Errorf("log = %q, want empty", out.String())
				}
				return
			}
			if !strings.Contains(out.String(), tc.wantLog) {
				t.Errorf("log = %q, want %q", out.String(), tc.wantLog)
			}
		})
	}
}

func TestRunRecordsHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"entity_id":"light.a","state":"on"}`)
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	captureLog(t)

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
	if err := run(cfg, []string{"state", "light.a"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 1 || history[0] != "light.a" {
		t.Errorf("history = %v, want [light.a]", history)
	}
}
