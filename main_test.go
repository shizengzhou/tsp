package main

import (
	"strings"
	"testing"
)

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run([]string{"nope"})
	if err == nil || !strings.Contains(err.Error(), `unknown command "nope"`) {
		t.Fatalf("run(nope) error = %v, want unknown command", err)
	}
}

func TestRunRejectsUnknownOmzCommand(t *testing.T) {
	err := run([]string{"omz", "nope"})
	if err == nil || !strings.Contains(err.Error(), `unknown omz command "nope"`) {
		t.Fatalf("run(omz nope) error = %v, want unknown omz command", err)
	}
}

func TestRunThemesRejectsExtraArgument(t *testing.T) {
	err := run([]string{"omz", "themes", "extra"})
	if err == nil || !strings.Contains(err.Error(), "unexpected argument") {
		t.Fatalf("run(omz themes extra) error = %v, want unexpected argument", err)
	}
}

func TestRunThemesRejectsRemovedPlainFlag(t *testing.T) {
	err := run([]string{"omz", "themes", "--plain"})
	if err == nil {
		t.Fatal("run(omz themes --plain) error = nil, want flag error")
	}
}
