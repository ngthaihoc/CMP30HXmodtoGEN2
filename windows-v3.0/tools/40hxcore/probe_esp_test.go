package hxcore

import (
	"errors"
	"strings"
	"testing"
)

func TestMountESPIgnoresOccupiedDriveWithEFIDirectory(t *testing.T) {
	previousExists, previousRun := espPathExists, espRunOut
	t.Cleanup(func() {
		espPathExists, espRunOut = previousExists, previousRun
	})
	mounted := false
	espPathExists = func(path string) bool {
		switch strings.ToUpper(path) {
		case "Y:\\", "Y:\\EFI":
			return true // A removable drive, not the system ESP.
		case "X:\\EFI":
			return mounted
		}
		return false
	}
	var commands []string
	espRunOut = func(name string, args ...string) (string, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if len(args) == 2 && args[0] == "X:" && args[1] == "/S" {
			mounted = true
		}
		return "", nil
	}
	if letter := MountESP(); letter != "X" {
		t.Fatalf("expected a newly mounted system ESP on X, got %q", letter)
	}
	if len(commands) != 1 || commands[0] != "mountvol.exe X: /S" {
		t.Fatalf("unexpected mount commands: %v", commands)
	}
}

func TestMountESPRejectsFailedMount(t *testing.T) {
	previousExists, previousRun := espPathExists, espRunOut
	t.Cleanup(func() {
		espPathExists, espRunOut = previousExists, previousRun
	})
	espPathExists = func(path string) bool {
		return strings.EqualFold(path, "Y:\\EFI")
	}
	espRunOut = func(name string, args ...string) (string, error) {
		return "", errors.New("mount denied")
	}
	if letter := MountESP(); letter != "" {
		t.Fatalf("failed mount should return no drive, got %q", letter)
	}
}
