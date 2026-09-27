package hxcore

// svchelper.go — Shared helpers for Windows Service Manager and COM Task Scheduler
// Eliminates duplicated boilerplate across link.go, service.go, state.go, uninstall_ops.go.

import (
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows/svc/mgr"
)

// Service state/start-type string constants (Fix 3: replace magic strings with typed constants)
type SvcStateString string
type SvcStartString string

const (
	SvcStateRunning         SvcStateString = "RUNNING"
	SvcStateStopped         SvcStateString = "STOPPED"
	SvcStateStartPending    SvcStateString = "START_PENDING"
	SvcStateStopPending     SvcStateString = "STOP_PENDING"
	SvcStatePaused          SvcStateString = "PAUSED"
	SvcStatePausePending    SvcStateString = "PAUSE_PENDING"
	SvcStateContinuePending SvcStateString = "CONTINUE_PENDING"
	SvcStateUnknown         SvcStateString = "UNKNOWN"

	SvcStartAuto     SvcStartString = "AUTO"
	SvcStartDemand   SvcStartString = "DEMAND"
	SvcStartDisabled SvcStartString = "DISABLED"
	SvcStartBoot     SvcStartString = "BOOT"
	SvcStartSystem   SvcStartString = "SYSTEM"
	SvcStartUnknown  SvcStartString = "UNKNOWN"
)

// Task Scheduler COM state constants (Fix 3: replace magic strings)
type TaskStateString string

const (
	TaskStateDisabled TaskStateString = "Disabled"
	TaskStateQueued   TaskStateString = "Queued"
	TaskStateReady    TaskStateString = "Ready"
	TaskStateRunning  TaskStateString = "Running"
	TaskStateUnknown  TaskStateString = "已注册"
)

// withService opens the Windows Service Manager, opens the named service,
// calls fn with the service handle, then closes both in the correct order.
// This eliminates the mgr.Connect / m.Disconnect / m.OpenService boilerplate
// duplicated across 6+ functions in link.go, service.go, state.go, uninstall_ops.go.
func withService(name string, fn func(*mgr.Service) error) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return err
	}
	defer s.Close()

	return fn(s)
}

// withTaskScheduler initialises COM, creates the Schedule.Service object,
// connects it, and calls fn with the root task folder IDispatch.
// Returns true if the COM setup succeeded and fn was called, false otherwise.
// The caller is responsible for releasing any COM objects it creates inside fn.
func withTaskScheduler(fn func(folder *ole.IDispatch) error) error {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Schedule.Service")
	if err != nil {
		return err
	}
	sched, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		unknown.Release()
		return err
	}
	defer unknown.Release()
	defer sched.Release()

	if _, err = oleutil.CallMethod(sched, "Connect"); err != nil {
		return err
	}

	folderRes, err := oleutil.CallMethod(sched, "GetFolder", "\\")
	if err != nil {
		return err
	}
	folder := folderRes.ToIDispatch()
	defer folder.Release()

	return fn(folder)
}
