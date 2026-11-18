package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hxcore "40hxcore"
	"golang.org/x/sys/windows/registry"
)

type startupTaskAction struct {
	Command   string `xml:"Command"`
	Arguments string `xml:"Arguments"`
}

type startupTaskXML struct {
	Actions []startupTaskAction `xml:"Actions>Exec"`
}

type startupRunKey interface {
	GetStringValue(string) (string, uint32, error)
	DeleteValue(string) error
	Close() error
}

type startupDependencies struct {
	run        func(string, ...string) (string, error)
	executable func() (string, error)
	taskExists func(string) (bool, error)
	openRunKey func() (startupRunKey, error)
	taskName   func() string
}

func defaultStartupDependencies() startupDependencies {
	return startupDependencies{
		run:        hxcore.RunOut,
		executable: os.Executable,
		taskExists: func(name string) (bool, error) {
			root := os.Getenv("SystemRoot")
			if root == "" {
				root = `C:\Windows`
			}
			_, err := os.Stat(filepath.Join(root, "System32", "Tasks", name))
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return err == nil, err
		},
		openRunKey: func() (startupRunKey, error) {
			return registry.OpenKey(registry.LOCAL_MACHINE,
				`SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
		},
		taskName: func() string {
			if prof, found := hxcore.FindGPUWithProfile(); found && prof.DeviceID == 0x1F0B {
				return taskName40HX
			}
			return taskName30HX
		},
	}
}

func startupExecutable(deps startupDependencies) (string, error) {
	exe, err := deps.executable()
	if err != nil {
		return "", fmt.Errorf("không thể xác định file thực thi: %w", err)
	}
	return filepath.Abs(exe)
}

func readStartupTask(deps startupDependencies, name string) (startupTaskAction, bool, error) {
	out, err := deps.run("schtasks.exe", "/query", "/tn", name, "/xml")
	if err != nil {
		exists, statErr := deps.taskExists(name)
		if statErr != nil {
			return startupTaskAction{}, false, fmt.Errorf("kiểm tra tác vụ %s: %w", name, statErr)
		}
		if !exists {
			return startupTaskAction{}, false, nil
		}
		return startupTaskAction{}, false, fmt.Errorf("đọc tác vụ %s: %w (%s)", name, err, strings.TrimSpace(out))
	}
	if end := strings.Index(out, "?>"); end >= 0 {
		out = out[end+2:]
	}
	var definition startupTaskXML
	if err := xml.Unmarshal([]byte(out), &definition); err != nil {
		return startupTaskAction{}, false, fmt.Errorf("định nghĩa tác vụ %s không hợp lệ: %w", name, err)
	}
	if len(definition.Actions) != 1 {
		return startupTaskAction{}, false, fmt.Errorf("tác vụ %s không có đúng một lệnh chạy", name)
	}
	return definition.Actions[0], true, nil
}

func ownedStartupTask(action startupTaskAction, exe string) bool {
	command := strings.Trim(action.Command, "\" ")
	return strings.EqualFold(filepath.Clean(command), filepath.Clean(exe)) &&
		strings.EqualFold(strings.TrimSpace(action.Arguments), "-gen2 -silent")
}

func removeOwnedStartupRunValue(deps startupDependencies, exe string) error {
	key, err := deps.openRunKey()
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mở khóa Run: %w", err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue("40HXGen2")
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("đọc khóa Run: %w", err)
	}
	ownedValue := fmt.Sprintf("\"%s\" -gen2 -silent", exe)
	if !strings.EqualFold(strings.TrimSpace(value), ownedValue) {
		return nil
	}
	if err := key.DeleteValue("40HXGen2"); err != nil {
		return fmt.Errorf("xóa khóa Run cũ: %w", err)
	}
	return nil
}

func registerAutostart(deps startupDependencies) error {
	exe, err := startupExecutable(deps)
	if err != nil {
		return err
	}
	name := deps.taskName()
	action, exists, err := readStartupTask(deps, name)
	if err != nil {
		return err
	}
	if exists && !ownedStartupTask(action, exe) {
		return fmt.Errorf("tác vụ %s thuộc lệnh khác; giữ nguyên", name)
	}
	args := []string{"/create", "/tn", name, "/tr", fmt.Sprintf("\"%s\" -gen2 -silent", exe),
		"/sc", "onlogon", "/ru", "SYSTEM", "/delay", "0000:10", "/rl", "highest"}
	if exists {
		args = append(args, "/f")
	}
	if out, err := deps.run("schtasks.exe", args...); err != nil {
		return fmt.Errorf("tạo tác vụ %s thất bại: %w (%s)", name, err, strings.TrimSpace(out))
	}
	action, exists, err = readStartupTask(deps, name)
	if err != nil {
		return err
	}
	if !exists || !ownedStartupTask(action, exe) {
		return fmt.Errorf("tác vụ %s không đúng sau khi đăng ký", name)
	}
	return removeOwnedStartupRunValue(deps, exe)
}

func unregisterAutostart(deps startupDependencies) error {
	exe, err := startupExecutable(deps)
	if err != nil {
		return err
	}
	for _, name := range []string{taskName30HX, taskName40HX} {
		action, exists, err := readStartupTask(deps, name)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if !ownedStartupTask(action, exe) {
			return fmt.Errorf("tác vụ %s thuộc lệnh khác; giữ nguyên", name)
		}
		if out, err := deps.run("schtasks.exe", "/delete", "/tn", name, "/f"); err != nil {
			return fmt.Errorf("xóa tác vụ %s thất bại: %w (%s)", name, err, strings.TrimSpace(out))
		}
		if _, stillExists, err := readStartupTask(deps, name); err != nil {
			return err
		} else if stillExists {
			return fmt.Errorf("tác vụ %s vẫn tồn tại sau khi xóa", name)
		}
	}
	return removeOwnedStartupRunValue(deps, exe)
}
