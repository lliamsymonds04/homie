package main

import (
	"slices"
	"testing"
)

func TestNoArgsCompletion(t *testing.T) {
	args := Complete(nil, nil)
	if args == nil {
		t.Error("No args should return command options")
	}
}

func TestComplete(t *testing.T) {
	history := []string{"light.desk", "switch.plug", "light.hall"}

	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"no args", nil, commands},
		{"after the entity", []string{"on", "light.desk"}, nil},
		{"after the colour", []string{"light", "light.desk", "red"}, nil},
		{"list domains", []string{"list"}, []string{"light", "switch"}},
		{"unknown command", []string{"nope"}, nil},
		{"hidden command", []string{"_complete"}, nil},
		{"shell for completion", []string{"completion"}, []string{"fish", "bash"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Complete(c.args, history); !slices.Equal(got, c.want) {
				t.Errorf("Complete(%v, %v) = %v, want %v", c.args, history, got, c.want)
			}
		})
	}
}
