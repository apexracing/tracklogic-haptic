package hpr

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	rimTypeHID     = 2
	ridiDeviceName = 0x20000007
	ridiDeviceInfo = 0x2000000B

	genericRead         = 0x80000000
	genericWrite        = 0x40000000
	fileShareRead       = 0x00000001
	fileShareWrite      = 0x00000002
	openExisting        = 0x00000003
	fileAttributeNormal = 0x00000080
)

type windowsDeviceScanner struct{}

type rawInputDeviceList struct {
	hDevice windows.Handle
	Type    uint32
}

type rawInputDeviceInfo struct {
	Size uint32
	Type uint32
	Data [24]byte
}

type rawDeviceInfoHID struct {
	VendorID      uint32
	ProductID     uint32
	VersionNumber uint32
	UsagePage     uint16
	Usage         uint16
}

type hidTransport struct {
	mu     sync.Mutex
	handle windows.Handle
}

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modHid      = windows.NewLazySystemDLL("hid.dll")

	procGetRawInputDeviceList     = modUser32.NewProc("GetRawInputDeviceList")
	procGetRawInputDeviceInfoW    = modUser32.NewProc("GetRawInputDeviceInfoW")
	procCreateFileW               = modKernel32.NewProc("CreateFileW")
	procHidDSetFeature            = modHid.NewProc("HidD_SetFeature")
	procHidDGetManufacturerString = modHid.NewProc("HidD_GetManufacturerString")
	procHidDGetProductString      = modHid.NewProc("HidD_GetProductString")
)

func (windowsDeviceScanner) ScanDevices() ([]DeviceInfo, error) {
	rawDevices, err := getRawInputDeviceList()
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(rawDevices))
	for _, d := range rawDevices {
		if d.Type != rimTypeHID {
			continue
		}

		hidInfo, err := getRawInputDeviceInfoHID(d.hDevice)
		if err != nil {
			continue
		}
		deviceName, err := getRawInputDeviceName(d.hDevice)
		if err != nil {
			continue
		}

		manufacturer, product, friendlyName := readFriendlyName(deviceName)
		if friendlyName == "" {
			friendlyName = getFallbackFriendlyName(deviceName)
		}

		devices = append(devices, DeviceInfo{
			DevicePath:    deviceName,
			FriendlyName:  friendlyName,
			Manufacturer:  manufacturer,
			Product:       product,
			VendorID:      hidInfo.VendorID,
			ProductID:     hidInfo.ProductID,
			VersionNumber: hidInfo.VersionNumber,
			UsagePage:     hidInfo.UsagePage,
			Usage:         hidInfo.Usage,
		})
	}
	return devices, nil
}

func openHIDTransport(info DeviceInfo) (Transport, error) {
	handle, err := createFile(info.DevicePath)
	if err != nil {
		return nil, err
	}
	return &hidTransport{handle: handle}, nil
}

func (t *hidTransport) SetFeature(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.handle == 0 {
		return ErrDeviceClosed
	}

	ret, _, callErr := procHidDSetFeature.Call(
		uintptr(t.handle),
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)
	if ret == 0 {
		if callErr != syscall.Errno(0) {
			return fmt.Errorf("HidD_SetFeature failed: len=%d data=% X: %w", len(data), data, callErr)
		}
		return fmt.Errorf("HidD_SetFeature failed: len=%d data=% X", len(data), data)
	}
	return nil
}

func (t *hidTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.handle == 0 {
		return nil
	}

	err := windows.CloseHandle(t.handle)
	t.handle = 0
	return err
}

func getRawInputDeviceList() ([]rawInputDeviceList, error) {
	var numDevices uint32
	ret, _, _ := procGetRawInputDeviceList.Call(0, uintptr(unsafe.Pointer(&numDevices)), unsafe.Sizeof(rawInputDeviceList{}))
	if ret == 0xFFFFFFFF {
		return nil, fmt.Errorf("GetRawInputDeviceList failed to get count")
	}
	if numDevices == 0 {
		return nil, nil
	}

	devices := make([]rawInputDeviceList, numDevices)
	ret, _, _ = procGetRawInputDeviceList.Call(
		uintptr(unsafe.Pointer(&devices[0])),
		uintptr(unsafe.Pointer(&numDevices)),
		unsafe.Sizeof(rawInputDeviceList{}),
	)
	if ret == 0xFFFFFFFF {
		return nil, fmt.Errorf("GetRawInputDeviceList failed to enumerate")
	}
	return devices[:numDevices], nil
}

func getRawInputDeviceName(hDevice windows.Handle) (string, error) {
	var size uint32
	ret, _, _ := procGetRawInputDeviceInfoW.Call(
		uintptr(hDevice), ridiDeviceName, 0, uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0xFFFFFFFF {
		return "", fmt.Errorf("GetRawInputDeviceInfoW failed to get size")
	}
	if size == 0 {
		return "", nil
	}

	buf := make([]uint16, size)
	ret, _, _ = procGetRawInputDeviceInfoW.Call(
		uintptr(hDevice), ridiDeviceName,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0xFFFFFFFF {
		return "", fmt.Errorf("GetRawInputDeviceInfoW failed to get name")
	}
	return windows.UTF16ToString(buf), nil
}

func getRawInputDeviceInfoHID(hDevice windows.Handle) (*rawDeviceInfoHID, error) {
	var info rawInputDeviceInfo
	info.Size = uint32(unsafe.Sizeof(info))
	size := info.Size

	ret, _, callErr := procGetRawInputDeviceInfoW.Call(
		uintptr(hDevice), ridiDeviceInfo,
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0xFFFFFFFF {
		if callErr != syscall.Errno(0) {
			return nil, fmt.Errorf("GetRawInputDeviceInfoW (DEVICEINFO) failed: %w", callErr)
		}
		return nil, fmt.Errorf("GetRawInputDeviceInfoW (DEVICEINFO) failed")
	}

	hid := *(*rawDeviceInfoHID)(unsafe.Pointer(&info.Data[0]))
	return &hid, nil
}

func createFile(devicePath string) (windows.Handle, error) {
	pathPtr, err := windows.UTF16PtrFromString(devicePath)
	if err != nil {
		return 0, err
	}

	handle, _, callErr := procCreateFileW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		genericRead|genericWrite,
		fileShareRead|fileShareWrite,
		0,
		openExisting,
		fileAttributeNormal,
		0,
	)

	if handle == 0 || handle == ^uintptr(0) {
		if callErr != syscall.Errno(0) {
			return 0, fmt.Errorf("CreateFileW failed: devicePath=%s: %w", devicePath, callErr)
		}
		return 0, fmt.Errorf("CreateFileW failed: devicePath=%s", devicePath)
	}
	return windows.Handle(handle), nil
}

func readFriendlyName(devicePath string) (manufacturer, product, friendlyName string) {
	handle, err := createFile(devicePath)
	if err != nil {
		return "", "", ""
	}
	defer windows.CloseHandle(handle)

	manufacturer = readHIDString(handle, procHidDGetManufacturerString)
	product = readHIDString(handle, procHidDGetProductString)
	switch {
	case manufacturer != "" && product != "":
		friendlyName = strings.TrimSpace(manufacturer + " " + product)
	case product != "":
		friendlyName = product
	default:
		friendlyName = manufacturer
	}
	return manufacturer, product, friendlyName
}

func readHIDString(handle windows.Handle, proc *windows.LazyProc) string {
	buf := make([]uint16, 256)
	ret, _, _ := proc.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)*2),
	)
	if ret == 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf))
}

func getFallbackFriendlyName(path string) string {
	upper := strings.ToUpper(path)
	if !strings.Contains(upper, "VID_3670") && !strings.Contains(upper, "VID_0483") {
		return ""
	}

	for _, part := range strings.Split(path, "#") {
		if strings.HasPrefix(strings.ToUpper(part), "VID_") {
			return part
		}
	}
	return ""
}
