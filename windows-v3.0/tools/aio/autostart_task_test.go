package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

const testAIOPath = `C:\Program Files\40HXUnlock\UnlockAIO.exe`

type fakeStartupRunKey struct {
	value   string
	deleted bool
}

func (k *fakeStartupRunKey) GetStringValue(name string) (string, uint32, error) {
	if name != "40HXGen2" || k.value == "" {
		return "", 0, registry.ErrNotExist
	}
	return k.value, 1, nil
}

func (k *fakeStartupRunKey) DeleteValue(name string) error {
	k.deleted = true
	k.value = ""
	return nil
}

func (k *fakeStartupRunKey) Close() error { return nil }

type fakeStartupScheduler struct {
	tasks       map[string]startupTaskAction
	created     []string
	deleted     []string
	deleteError error
}

func (f *fakeStartupScheduler) run(_ string, args ...string) (string, error) {
	switch args[0] {
	case "/query":
		action, ok := f.tasks[args[2]]
		if !ok {
			return "", errors.New("task not found")
		}
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?><Task><Actions><Exec><Command>%s</Command><Arguments>%s</Arguments></Exec></Actions></Task>`, action.Command, action.Arguments), nil
	case "/create":
		f.created = append(f.created, strings.Join(args, " "))
		name := argumentAfter(args, "/tn")
		if _, exists := f.tasks[name]; exists && !containsArgument(args, "/f") {
			return "", errors.New("task already exists")
		}
		f.tasks[name] = startupTaskAction{Command: testAIOPath, Arguments: "-gen2 -silent"}
		return "created", nil
	case "/delete":
		if f.deleteError != nil {
			return "", f.deleteError
		}
		name := argumentAfter(args, "/tn")
		f.deleted = append(f.deleted, name)
		delete(f.tasks, name)
		return "deleted", nil
	}
	return "", fmt.Errorf("unexpected schtasks command: %v", args)
}

func argumentAfter(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func containsArgument(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func startupTestDeps(f *fakeStartupScheduler, key *fakeStartupRunKey) startupDependencies {
	return startupDependencies{
		run:        f.run,
		executable: func() (string, error) { return testAIOPath, nil },
		taskExists: func(name string) (bool, error) { _, ok := f.tasks[name]; return ok, nil },
		openRunKey: func() (startupRunKey, error) { return key, nil },
		taskName:   func() string { return taskName30HX },
	}
}

func TestRegisterAutostartDoesNotOverwriteUnrelatedTask(t *testing.T) {
	f := &fakeStartupScheduler{tasks: map[string]startupTaskAction{
		taskName30HX: {Command: `C:\Other\app.exe`, Arguments: "-gen2 -silent"},
	}}
	key := &fakeStartupRunKey{}
	if err := registerAutostart(startupTestDeps(f, key)); err == nil {
		t.Fatal("expected task ownership conflict")
	}
	if len(f.created) != 0 || key.deleted {
		t.Fatalf("unrelated task or Run value changed: %+v %+v", f, key)
	}
}

func TestRegisterAutostartUsesSchedulerOnlyAndCleansOwnedLegacyRunValue(t *testing.T) {
	f := &fakeStartupScheduler{tasks: map[string]startupTaskAction{}}
	key := &fakeStartupRunKey{value: `"C:\Program Files\40HXUnlock\UnlockAIO.exe" -gen2 -silent`}
	if err := registerAutostart(startupTestDeps(f, key)); err != nil {
		t.Fatal(err)
	}
	if len(f.created) != 1 || strings.Contains(f.created[0], " /f") {
		t.Fatalf("new task used forced replacement: %v", f.created)
	}
	if !key.deleted || !ownedStartupTask(f.tasks[taskName30HX], testAIOPath) {
		t.Fatalf("owned task or legacy Run cleanup missing: %+v %+v", f, key)
	}
}

func TestUnregisterAutostartReportsDeletionFailureAndKeepsRunValue(t *testing.T) {
	f := &fakeStartupScheduler{tasks: map[string]startupTaskAction{
		taskName30HX: {Command: testAIOPath, Arguments: "-gen2 -silent"},
	}, deleteError: errors.New("access denied")}
	key := &fakeStartupRunKey{value: `"C:\Program Files\40HXUnlock\UnlockAIO.exe" -gen2 -silent`}
	if err := unregisterAutostart(startupTestDeps(f, key)); err == nil {
		t.Fatal("expected task deletion failure")
	}
	if key.deleted || len(f.deleted) != 0 {
		t.Fatalf("cleanup continued after scheduler failure: %+v %+v", f, key)
	}
}

func TestUnregisterAutostartPreservesOtherTaskAndRunValue(t *testing.T) {
	f := &fakeStartupScheduler{tasks: map[string]startupTaskAction{
		taskName30HX: {Command: `C:\Other\UnlockAIO.exe`, Arguments: "-gen2 -silent"},
	}}
	key := &fakeStartupRunKey{value: `"C:\Other\UnlockAIO.exe" -gen2 -silent`}
	if err := unregisterAutostart(startupTestDeps(f, key)); err == nil {
		t.Fatal("expected ownership conflict")
	}
	if len(f.deleted) != 0 || key.deleted {
		t.Fatalf("unrelated entries changed: %+v %+v", f, key)
	}
}

func TestUnregisterAutostartIsRepeatable(t *testing.T) {
	f := &fakeStartupScheduler{tasks: map[string]startupTaskAction{
		taskName30HX: {Command: testAIOPath, Arguments: "-gen2 -silent"},
	}}
	key := &fakeStartupRunKey{value: `"C:\Program Files\40HXUnlock\UnlockAIO.exe" -gen2 -silent`}
	deps := startupTestDeps(f, key)
	for i := 0; i < 2; i++ {
		if err := unregisterAutostart(deps); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.deleted) != 1 || !key.deleted {
		t.Fatalf("owned cleanup failed: %+v %+v", f, key)
	}
}
