//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const (
	cuAttrMultiprocessorCount    = 16
	cuAttrComputeCapabilityMajor = 75
	cuAttrComputeCapabilityMinor = 76
	loadLibrarySearchSystem32    = 0x00000800
)

var (
	modKernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemDirectory = modKernel32.NewProc("GetSystemDirectoryW")
	procLoadLibraryExW     = modKernel32.NewProc("LoadLibraryExW")
)

func systemCudaDLLPath(systemDir string) (string, error) {
	trimmed := strings.TrimSpace(systemDir)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty System32 path", ErrCudaUnavailable)
	}
	return filepath.Join(trimmed, "nvcuda.dll"), nil
}

func resolveSystemDir() (string, error) {
	buf := make([]uint16, 260)
	n, _, err := procGetSystemDirectory.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return "", fmt.Errorf("%w: GetSystemDirectoryW failed: %v", ErrCudaUnavailable, err)
	}
	if int(n) > len(buf) {
		buf = make([]uint16, int(n)+1)
		n, _, err = procGetSystemDirectory.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 || int(n) > len(buf) {
			return "", fmt.Errorf("%w: GetSystemDirectoryW failed: %v", ErrCudaUnavailable, err)
		}
	}
	return syscall.UTF16ToString(buf[:n]), nil
}

func loadSystemCudaDLL() (*syscall.DLL, error) {
	sysDir, err := resolveSystemDir()
	if err != nil {
		return nil, err
	}
	dllPath, err := systemCudaDLLPath(sysDir)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(dllPath); statErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrCudaUnavailable, statErr)
	}
	ptr, err := syscall.UTF16PtrFromString(dllPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCudaUnavailable, err)
	}
	handle, _, callErr := procLoadLibraryExW.Call(uintptr(unsafe.Pointer(ptr)), 0, uintptr(loadLibrarySearchSystem32))
	if handle == 0 {
		return nil, fmt.Errorf("%w: LoadLibraryExW failed: %v", ErrCudaUnavailable, callErr)
	}
	return &syscall.DLL{Name: dllPath, Handle: syscall.Handle(handle)}, nil
}

// RealCudaDriver invokes Windows System32 nvcuda.dll dynamically with zero CGO or Toolkit install.
type RealCudaDriver struct {
	dll                *syscall.DLL
	cuInit             *syscall.Proc
	cuDriverGetVersion *syscall.Proc
	cuDeviceByBusID    *syscall.Proc
	cuDeviceGetName    *syscall.Proc
	cuDeviceGetAttr    *syscall.Proc
	cuCtxCreate        *syscall.Proc
	cuCtxDestroy       *syscall.Proc
	cuMemHostRegister  *syscall.Proc
	cuMemHostUnreg     *syscall.Proc
	cuMemAlloc         *syscall.Proc
	cuMemFree          *syscall.Proc
	cuMemcpyHtoD       *syscall.Proc
	cuMemcpyDtoH       *syscall.Proc
	cuCtxSynchronize   *syscall.Proc
	cuModuleLoadData   *syscall.Proc
	cuModuleUnload     *syscall.Proc
	cuModuleGetFunc    *syscall.Proc
	cuLaunchKernel     *syscall.Proc
}

// NewRealCudaDriver constructs an unbound nvcuda.dll driver wrapper; Init() loads exclusively from System32.
func NewRealCudaDriver() *RealCudaDriver {
	return &RealCudaDriver{}
}

func checkCU(name string, r1 uintptr) error {
	code := uint32(r1)
	if code != 0 {
		return fmt.Errorf("%s returned CUresult %d", name, code)
	}
	return nil
}

func cstr(s string) []byte {
	b := make([]byte, len(s)+1)
	copy(b, s)
	return b
}

func (d *RealCudaDriver) Init() error {
	dll, err := loadSystemCudaDLL()
	if err != nil {
		return err
	}
	procs := []struct {
		target **syscall.Proc
		name   string
	}{
		{&d.cuInit, "cuInit"},
		{&d.cuDriverGetVersion, "cuDriverGetVersion"},
		{&d.cuDeviceByBusID, "cuDeviceGetByPCIBusId"},
		{&d.cuDeviceGetName, "cuDeviceGetName"},
		{&d.cuDeviceGetAttr, "cuDeviceGetAttribute"},
		{&d.cuCtxCreate, "cuCtxCreate_v2"},
		{&d.cuCtxDestroy, "cuCtxDestroy_v2"},
		{&d.cuMemHostRegister, "cuMemHostRegister_v2"},
		{&d.cuMemHostUnreg, "cuMemHostUnregister"},
		{&d.cuMemAlloc, "cuMemAlloc_v2"},
		{&d.cuMemFree, "cuMemFree_v2"},
		{&d.cuMemcpyHtoD, "cuMemcpyHtoD_v2"},
		{&d.cuMemcpyDtoH, "cuMemcpyDtoH_v2"},
		{&d.cuCtxSynchronize, "cuCtxSynchronize"},
		{&d.cuModuleLoadData, "cuModuleLoadData"},
		{&d.cuModuleUnload, "cuModuleUnload"},
		{&d.cuModuleGetFunc, "cuModuleGetFunction"},
		{&d.cuLaunchKernel, "cuLaunchKernel"},
	}
	for _, p := range procs {
		proc, ferr := dll.FindProc(p.name)
		if ferr != nil {
			return fmt.Errorf("%w: missing proc %s: %v", ErrCudaUnavailable, p.name, ferr)
		}
		*p.target = proc
	}
	d.dll = dll
	r1, _, _ := d.cuInit.Call(0)
	return checkCU("cuInit", r1)
}

func (d *RealCudaDriver) DriverVersion() (int, error) {
	var ver int32
	r1, _, _ := d.cuDriverGetVersion.Call(uintptr(unsafe.Pointer(&ver)))
	if err := checkCU("cuDriverGetVersion", r1); err != nil {
		return 0, err
	}
	return int(ver), nil
}

func (d *RealCudaDriver) DeviceByPCIBusID(bdf string) (int32, error) {
	var dev int32
	busBytes := cstr(bdf)
	r1, _, _ := d.cuDeviceByBusID.Call(uintptr(unsafe.Pointer(&dev)), uintptr(unsafe.Pointer(&busBytes[0])))
	runtime.KeepAlive(busBytes)
	if err := checkCU("cuDeviceGetByPCIBusId", r1); err != nil {
		return 0, err
	}
	return dev, nil
}

func (d *RealCudaDriver) DeviceName(dev int32) (string, error) {
	buf := make([]byte, 256)
	r1, _, _ := d.cuDeviceGetName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(dev))
	if err := checkCU("cuDeviceGetName", r1); err != nil {
		return "", err
	}
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}

func (d *RealCudaDriver) ComputeCapability(dev int32) (int, int, error) {
	var maj, min int32
	r1, _, _ := d.cuDeviceGetAttr.Call(uintptr(unsafe.Pointer(&maj)), cuAttrComputeCapabilityMajor, uintptr(dev))
	if err := checkCU("cuDeviceGetAttribute(major)", r1); err != nil {
		return 0, 0, err
	}
	r2, _, _ := d.cuDeviceGetAttr.Call(uintptr(unsafe.Pointer(&min)), cuAttrComputeCapabilityMinor, uintptr(dev))
	if err := checkCU("cuDeviceGetAttribute(minor)", r2); err != nil {
		return 0, 0, err
	}
	return int(maj), int(min), nil
}

func (d *RealCudaDriver) CreateContext(dev int32) (uintptr, error) {
	var ctx uintptr
	r1, _, _ := d.cuCtxCreate.Call(uintptr(unsafe.Pointer(&ctx)), 0, uintptr(dev))
	if err := checkCU("cuCtxCreate_v2", r1); err != nil {
		return 0, err
	}
	return ctx, nil
}

func (d *RealCudaDriver) DestroyContext(ctx uintptr) error {
	r1, _, _ := d.cuCtxDestroy.Call(ctx)
	return checkCU("cuCtxDestroy_v2", r1)
}

func (d *RealCudaDriver) RegisterHost(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	r1, _, _ := d.cuMemHostRegister.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	runtime.KeepAlive(buf)
	return checkCU("cuMemHostRegister_v2", r1)
}

func (d *RealCudaDriver) UnregisterHost(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	r1, _, _ := d.cuMemHostUnreg.Call(uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(buf)
	return checkCU("cuMemHostUnregister", r1)
}

func (d *RealCudaDriver) AllocDevice(bytes uintptr) (uint64, error) {
	var dptr uint64
	r1, _, _ := d.cuMemAlloc.Call(uintptr(unsafe.Pointer(&dptr)), bytes)
	if err := checkCU("cuMemAlloc_v2", r1); err != nil {
		return 0, err
	}
	return dptr, nil
}

func (d *RealCudaDriver) FreeDevice(dptr uint64) error {
	r1, _, _ := d.cuMemFree.Call(uintptr(dptr))
	return checkCU("cuMemFree_v2", r1)
}

func (d *RealCudaDriver) MemcpyHtoD(dst uint64, src []byte) error {
	if len(src) == 0 {
		return nil
	}
	r1, _, _ := d.cuMemcpyHtoD.Call(uintptr(dst), uintptr(unsafe.Pointer(&src[0])), uintptr(len(src)))
	runtime.KeepAlive(src)
	return checkCU("cuMemcpyHtoD_v2", r1)
}

func (d *RealCudaDriver) MemcpyDtoH(dst []byte, src uint64) error {
	if len(dst) == 0 {
		return nil
	}
	r1, _, _ := d.cuMemcpyDtoH.Call(uintptr(unsafe.Pointer(&dst[0])), uintptr(src), uintptr(len(dst)))
	runtime.KeepAlive(dst)
	return checkCU("cuMemcpyDtoH_v2", r1)
}

func (d *RealCudaDriver) Synchronize() error {
	r1, _, _ := d.cuCtxSynchronize.Call()
	return checkCU("cuCtxSynchronize", r1)
}

func (d *RealCudaDriver) LoadModule(ptx string) (uintptr, error) {
	var mod uintptr
	ptxBytes := cstr(ptx)
	r1, _, _ := d.cuModuleLoadData.Call(uintptr(unsafe.Pointer(&mod)), uintptr(unsafe.Pointer(&ptxBytes[0])))
	runtime.KeepAlive(ptxBytes)
	if err := checkCU("cuModuleLoadData", r1); err != nil {
		return 0, err
	}
	return mod, nil
}

func (d *RealCudaDriver) UnloadModule(mod uintptr) error {
	r1, _, _ := d.cuModuleUnload.Call(mod)
	return checkCU("cuModuleUnload", r1)
}

func (d *RealCudaDriver) GetFunction(mod uintptr, name string) (uintptr, error) {
	var fn uintptr
	fnBytes := cstr(name)
	r1, _, _ := d.cuModuleGetFunc.Call(uintptr(unsafe.Pointer(&fn)), mod, uintptr(unsafe.Pointer(&fnBytes[0])))
	runtime.KeepAlive(fnBytes)
	if err := checkCU("cuModuleGetFunction", r1); err != nil {
		return 0, err
	}
	return fn, nil
}

func (d *RealCudaDriver) LaunchSAXPY(fn uintptr, n uint32, dA, dB, dOut uint64, alpha float32) error {
	const blockX uint32 = 256
	gridX := (n + blockX - 1) / blockX
	pA := dA
	pB := dB
	pOut := dOut
	aVal := alpha
	nVal := n
	params := [5]uintptr{
		uintptr(unsafe.Pointer(&pA)),
		uintptr(unsafe.Pointer(&pB)),
		uintptr(unsafe.Pointer(&pOut)),
		uintptr(unsafe.Pointer(&aVal)),
		uintptr(unsafe.Pointer(&nVal)),
	}
	r1, _, _ := d.cuLaunchKernel.Call(
		fn,
		uintptr(gridX), 1, 1,
		uintptr(blockX), 1, 1,
		0,
		0,
		uintptr(unsafe.Pointer(&params[0])),
		0,
	)
	runtime.KeepAlive(pA)
	runtime.KeepAlive(pB)
	runtime.KeepAlive(pOut)
	runtime.KeepAlive(aVal)
	runtime.KeepAlive(nVal)
	runtime.KeepAlive(params)
	return checkCU("cuLaunchKernel", r1)
}
