package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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

func TestClearHistory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := RecordHistory("one")
	if err != nil {
		t.Fatal(err)
	}

	err = ClearHistory()
	if err != nil {
		t.Fatal(err)
	}

	// check again once its empty to make sure it doesnt error
	err = ClearHistory()
	if err != nil {
		t.Fatal(err)
	}

	history, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("LoadHistory = %+v, want %+v", history, []string{})
	}
}
