package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer bad" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer bad")
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL, HomeAssistantToken: "bad"}

	if _, err := Get(cfg, "states"); err == nil || !strings.Contains(err.Error(), "token rejected") {
		t.Fatalf("get: want token rejected error, got %v", err)
	}

	if _, err := Set(cfg, "services/light/turn_on", "light.kitchen", nil); err == nil {
		t.Fatal("set: want error for 401, got nil")
	}
}

func TestGetStatusErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string // error substring
	}{
		{"not found", http.StatusNotFound, "", "not found"},
		{"server error", http.StatusInternalServerError, "boom", "500 Internal Server Error: boom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
			_, err := Get(cfg, "states/light.a")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Get = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestApiAuthSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer good" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer good")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL, HomeAssistantToken: "good"}

	if _, err := Get(cfg, "states"); err != nil {
		t.Fatalf("get: want nil, got %v", err)
	}

	if _, err := Set(cfg, "services/light/turn_on", "light.kitchen", nil); err != nil {
		t.Fatalf("set: want nil, got %v", err)
	}
}

func TestSetEntityPath(t *testing.T) {
	cases := []struct {
		verb   string
		entity string
		want   string
	}{
		{"on", "light.desk_lamp", "/api/services/light/turn_on"},
		{"off", "switch.plug", "/api/services/switch/turn_off"},
		{"toggle", "light.desk_lamp", "/api/services/light/toggle"},
		{"open_cover", "cover.garage", "/api/services/cover/open_cover"},
	}
	for _, tc := range cases {
		t.Run(tc.verb+"_"+tc.entity, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.URL.Path
				io.WriteString(w, "[]")
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
			if err := SetEntity(cfg, tc.verb, tc.entity); err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSetInvalidEntity(t *testing.T) {
	for _, id := range []string{"", "invalid", ".lamp", "light."} {
		t.Run(id, func(t *testing.T) {
			called := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
			if err := SetEntity(cfg, "toggle", id); err == nil {
				t.Errorf("SetEntity(%q) = %v, want error", id, err)
			}
			if called {
				t.Errorf("SetEntity(%q) called, want not called", id)
			}
		})
	}
}

func TestTriggerAutomation(t *testing.T) {
	cases := []struct {
		entity   string
		wantPath string // empty means the request must never be made
	}{
		{"automation.morning", "/api/services/automation/trigger"},
		{"", ""},
		{"invalid", ""},
		{"automation.", ""},
		{".morning", ""},
		{"light.desk_lamp", ""},
	}

	for _, tc := range cases {
		t.Run(tc.entity, func(t *testing.T) {
			var method, path string
			var body map[string]string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				path = r.URL.Path
				if got := r.Header.Get("Authorization"); got != "Bearer good" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer good")
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode body: %v", err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "good"}
			err := TriggerAutomation(cfg, tc.entity)

			if tc.wantPath == "" {
				if err == nil {
					t.Errorf("TriggerAutomation(%q) = nil, want error", tc.entity)
				}
				if path != "" {
					t.Errorf("TriggerAutomation(%q) sent %s %s, want no request", tc.entity, method, path)
				}
				return
			}

			if err != nil {
				t.Fatalf("TriggerAutomation = %v, want nil", err)
			}
			if method != http.MethodPost {
				t.Errorf("method = %q, want %q", method, http.MethodPost)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
			if got := body["entity_id"]; got != tc.entity {
				t.Errorf("entity_id = %q, want %q", got, tc.entity)
			}
		})
	}
}

func TestGetEntities(t *testing.T) {
	cases := []struct {
		name string
		body string
		warn []Entity
	}{
		{
			name: "decodes states",
			body: `[{"entity_id":"light.desk_lamp","state":"on"}]`,
			warn: []Entity{{
				ID:    "light.desk_lamp",
				State: "on",
			}},
		},
		{
			name: "empty",
			body: `[]`,
			warn: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/states" {
					t.Errorf("path = %q, want %q", r.URL.Path, "/api/states")
				}
				if got := r.Header.Get("Authorization"); got != "Bearer good" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer good")
				}
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "good"}
			got, err := GetEntities(cfg)
			if err != nil {
				t.Fatalf("GetEntities = %v, want nil", err)
			}
			if !slices.Equal(got, tc.warn) {
				t.Errorf("got %v, want %v", got, tc.warn)
			}
		})
	}
}

func TestGetEntitiesBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `nope`)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "test"}

	if _, err := GetEntities(cfg); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"entity_id":"light.a","state":"on"},
                          {"entity_id":"light.b","state":"off"},
                          {"entity_id":"switch.c","state":"on"}]`)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags)
	}()

	if err := list(cfg, "light"); err != nil {
		t.Fatalf("list: %v", err)
	}

	var got []string
	if s := strings.TrimSpace(buf.String()); s != "" {
		for p := range strings.SplitSeq(s, "\n") {
			got = append(got, strings.TrimPrefix(p, "entity: "))
		}
	}

	want := []string{"light.a", "light.b"}
	if !slices.Equal(got, want) {
		t.Errorf("listed %v, want %v", got, want)
	}
}

func TestGetState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/states/light.a" {
			t.Errorf("request = %s %s, want GET /api/states/light.a", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer x" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer x")
		}
		io.WriteString(w, `{"entity_id":"light.a","state":"on"}`)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}

	got, err := GetState(cfg, "light.a")
	if err != nil {
		t.Fatalf("GetState = %v, want nil", err)
	}
	if got != "on" {
		t.Errorf("got %v, want on", got)
	}
}

func TestGetStateBadJson(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "nope")
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}

	_, err := GetState(cfg, "light.a")
	if err == nil {
		t.Errorf("GetState = nil, want error")
	}
}

func TestSetLightColour(t *testing.T) {
	var path string
	var body map[string]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
	if err := SetLightColour(cfg, "light.desk_lamp", "red"); err != nil {
		t.Fatalf("SetLightColour = %v, want nil", err)
	}

	if want := "/api/services/light/turn_on"; path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if got := body["entity_id"]; got != "light.desk_lamp" {
		t.Errorf("entity_id = %q, want %q", got, "light.desk_lamp")
	}
	if got := body["color_name"]; got != "red" {
		t.Errorf("color_name = %q, want %q", got, "red")
	}
}

func TestSetLightColourWrongDomain(t *testing.T) {
	cases := []struct {
		entity string
		want   string // error substring
	}{
		{"switch.desk_lamp", "not from the light domain"},
		{".desk_lamp", "not from the light domain"},
		{"light", "no domain"},
		{"light.a.b", "no domain"},
		{"", "no domain"},
	}

	for _, tc := range cases {
		t.Run(tc.entity, func(t *testing.T) {
			called := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}))
			defer srv.Close()

			cfg := Config{HomeAssistantEndpoint: srv.URL + "/api", HomeAssistantToken: "x"}
			err := SetLightColour(cfg, tc.entity, "red")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("SetLightColour(%q) = %v, want error containing %q", tc.entity, err, tc.want)
			}
			if called {
				t.Errorf("SetLightColour(%q) made a request, want none", tc.entity)
			}
		})
	}
}
