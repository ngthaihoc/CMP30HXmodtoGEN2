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
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService("NVDisplay.ContainerLocalSystem")
	if err != nil {
		return err
	}
	defer s.Close()

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
	_, err = RunOut("reg.exe", "add", `HKCR\Directory\Background\shellex\ContextMenuHandlers\NvCplDesktopContext`, "/ve", "/t", "REG_SZ", "/d", "{3D1975AF-48C6-4f8e-A182-BE0E08FA86A9}", "/f")
	return err
}

// ServiceInfo: 查询内核驱动服务。
func ServiceInfo(name string) (bool, string, string) {
	m, err := mgr.Connect()
	if err != nil {
		return false, "", ""
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return false, "", ""
	}
	defer s.Close()

	st, err := s.Query()
	stateStr := "UNKNOWN"
	if err == nil {
		switch st.State {
		case svc.Running:
			stateStr = "RUNNING"
		case svc.Stopped:
			stateStr = "STOPPED"
		case svc.StartPending:
			stateStr = "START_PENDING"
		case svc.StopPending:
			stateStr = "STOP_PENDING"
		case svc.Paused:
			stateStr = "PAUSED"
		case svc.PausePending:
			stateStr = "PAUSE_PENDING"
		case svc.ContinuePending:
			stateStr = "CONTINUE_PENDING"
		}
	}

	stype := "UNKNOWN"
	if conf, err := s.Config(); err == nil {
		switch conf.StartType {
		case mgr.StartManual:
			stype = "DEMAND"
		case mgr.StartAutomatic:
			stype = "AUTO"
		case mgr.StartDisabled:
			stype = "DISABLED"
		case windows.SERVICE_BOOT_START:
			stype = "BOOT"
		case windows.SERVICE_SYSTEM_START:
			stype = "SYSTEM"
		}
	}
	return true, stype, stateStr
}

// TaskInfo: 查询计划任务 (COM go-ole)。
func TaskInfo(name string) (bool, string, string) {
	if !taskXMLExists(name) {
		return false, "", ""
	}

	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("Schedule.Service")
	if err != nil {
		return true, "已注册", ""
	}
	sched, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return true, "已注册", ""
	}
	defer sched.Release()

	_, err = oleutil.CallMethod(sched, "Connect")
	if err != nil {
		return true, "已注册", ""
	}

	folderRes, err := oleutil.CallMethod(sched, "GetFolder", "\\")
	if err != nil {
		return true, "已注册", ""
	}
	folder := folderRes.ToIDispatch()
	defer folder.Release()

	taskRes, err := oleutil.CallMethod(folder, "GetTask", name)
	if err != nil {
		return true, "已注册", ""
	}
	task := taskRes.ToIDispatch()
	defer task.Release()

	status := "已注册"
	if stateRes, err := oleutil.GetProperty(task, "State"); err == nil {
		switch int(stateRes.Val) {
		case 1:
			status = "Disabled"
		case 2:
			status = "Queued"
		case 3:
			status = "Ready"
		case 4:
			status = "Running"
		}
	}

	lastResult := ""
	if resultRes, err := oleutil.GetProperty(task, "LastTaskResult"); err == nil {
		lastResult = fmt.Sprintf("%v", resultRes.Value())
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
