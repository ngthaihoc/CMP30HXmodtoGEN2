package hxcore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Gen2StatusPath: %ProgramData%\40HXUnlock\gen2_status.txt
func Gen2StatusPath() string {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "40HXUnlock", "gen2_status.txt")
}

// WriteGen2Status: Gen2 执行结果写入状态文件(供 40HXCheck 展示)。
func WriteGen2Status(text string) error {
	p := Gen2StatusPath()
	os.MkdirAll(filepath.Dir(p), 0o755)
	head := "==== 40HX Gen2 任务状态 " + time.Now().Format("2006-01-02 15:04:05") + " ====\n"
	return os.WriteFile(p, []byte(head+text+"\n"), 0o644)
}

// ReadGen2Status: 读状态文件; 不存在/读不到返回 ""
func ReadGen2Status() string {
	b, err := os.ReadFile(Gen2StatusPath())
	if err != nil {
		return ""
	}
	return string(b)
}

// EnsureNvidiaControlPanelHealthy: Đảm bảo service NVDisplay.ContainerLocalSystem chạy (Auto)
func EnsureNvidiaControlPanelHealthy() error {
	err := withService("NVDisplay.ContainerLocalSystem", func(s *mgr.Service) error {
		conf, err := s.Config()
		if err == nil && conf.StartType != mgr.StartAutomatic {
			conf.StartType = mgr.StartAutomatic
			s.UpdateConfig(conf)
		}

		st, err := s.Query()
		if err == nil && st.State != svc.Running {
			s.Start()
			for i := 0; i < 20; i++ {
				time.Sleep(150 * time.Millisecond)
				if st, err := s.Query(); err == nil && st.State == svc.Running {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	_, err = RunOut("reg.exe", "add", `HKCR\Directory\Background\shellex\ContextMenuHandlers\NvCplDesktopContext`, "/ve", "/t", "REG_SZ", "/d", "{3D1975AF-48C6-4f8e-A182-BE0E08FA86A9}", "/f")
	return err
}

// ServiceInfo: 查询内核驱动服务。
func ServiceInfo(name string) (bool, string, string) {
	var stateStr SvcStateString = SvcStateUnknown
	var stype SvcStartString = SvcStartUnknown

	err := withService(name, func(s *mgr.Service) error {
		st, err := s.Query()
		if err == nil {
			switch st.State {
			case svc.Running:
				stateStr = SvcStateRunning
			case svc.Stopped:
				stateStr = SvcStateStopped
			case svc.StartPending:
				stateStr = SvcStateStartPending
			case svc.StopPending:
				stateStr = SvcStateStopPending
			case svc.Paused:
				stateStr = SvcStatePaused
			case svc.PausePending:
				stateStr = SvcStatePausePending
			case svc.ContinuePending:
				stateStr = SvcStateContinuePending
			}
		}

		if conf, err := s.Config(); err == nil {
			switch conf.StartType {
			case mgr.StartManual:
				stype = SvcStartDemand
			case mgr.StartAutomatic:
				stype = SvcStartAuto
			case mgr.StartDisabled:
				stype = SvcStartDisabled
			case windows.SERVICE_BOOT_START:
				stype = SvcStartBoot
			case windows.SERVICE_SYSTEM_START:
				stype = SvcStartSystem
			}
		}
		return nil
	})
	if err != nil {
		return false, "", ""
	}
	return true, string(stype), string(stateStr)
}

// TaskInfo: 查询计划任务 (COM go-ole)。
func TaskInfo(name string) (bool, string, string) {
	if !taskXMLExists(name) {
		return false, "", ""
	}

	status := string(TaskStateUnknown)
	lastResult := ""

	err := withTaskScheduler(func(folder *ole.IDispatch) error {
		taskRes, err := oleutil.CallMethod(folder, "GetTask", name)
		if err != nil {
			// Task registered (XML exists) but COM can't enumerate it
			return nil
		}
		task := taskRes.ToIDispatch()
		defer task.Release()

		if stateRes, err := oleutil.GetProperty(task, "State"); err == nil {
			switch int(stateRes.Val) {
			case 1:
				status = string(TaskStateDisabled)
			case 2:
				status = string(TaskStateQueued)
			case 3:
				status = string(TaskStateReady)
			case 4:
				status = string(TaskStateRunning)
			}
		}

		if resultRes, err := oleutil.GetProperty(task, "LastTaskResult"); err == nil {
			lastResult = fmt.Sprintf("%v", resultRes.Value())
		}
		return nil
	})
	if err != nil {
		// COM init or connection failed — task XML exists so report registered
		return true, string(TaskStateUnknown), ""
	}
	return true, status, lastResult
}

// taskXMLExists:
func taskXMLExists(name string) bool {
	sysroot := os.Getenv("SystemRoot")
	if sysroot == "" {
		sysroot = `C:\Windows`
	}
	p := filepath.Join(sysroot, "System32", "Tasks", name)
	if _, e := os.Stat(p); e == nil {
		return true
	}
	return false
}
