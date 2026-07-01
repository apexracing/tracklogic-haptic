package hpr

import (
	"errors"
	"testing"
)

type fakeDriver struct {
	name     string
	match    bool
	describe func(DeviceInfo) DeviceInfo
	opened   int
	gotInfo  DeviceInfo
}

func (d *fakeDriver) Match(DeviceInfo) bool { return d.match }

func (d *fakeDriver) Describe(info DeviceInfo) DeviceInfo {
	if d.describe != nil {
		return d.describe(info)
	}
	return info
}

func (d *fakeDriver) Open(info DeviceInfo) (Device, error) {
	d.opened++
	d.gotInfo = info
	return stubDevice{info: info}, nil
}

// withScanner swaps scanDevicesImpl for the duration of the test,
// restoring the original on cleanup.
func withScanner(devices []DeviceInfo, err error) func() {
	orig := scanDevicesImpl
	scanDevicesImpl = func() ([]DeviceInfo, error) { return devices, err }
	return func() { scanDevicesImpl = orig }
}

func TestManager_ScanFiltersByDriverMatch(t *testing.T) {
	defer withScanner([]DeviceInfo{
		{DevicePath: "a"}, {DevicePath: "b"},
	}, nil)()
	m := NewManager(WithDrivers(&fakeDriver{name: "any", match: true}))
	got, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Scan returned %d devices, want 2", len(got))
	}
}

func TestManager_ScanDropsUnmatchedDevices(t *testing.T) {
	defer withScanner([]DeviceInfo{{DevicePath: "x"}}, nil)()
	m := NewManager(WithDrivers(&fakeDriver{name: "picky", match: false}))
	got, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("Scan returned %d devices, want 0", len(got))
	}
}

func TestManager_ScanDoesNotOpenDevices(t *testing.T) {
	d := &fakeDriver{name: "any", match: true}
	defer withScanner([]DeviceInfo{{DevicePath: "x"}}, nil)()
	m := NewManager(WithDrivers(d))
	_, _ = m.Scan()
	if d.opened != 0 {
		t.Fatalf("Scan should not open devices, got %d", d.opened)
	}
}

func TestManager_ScanCapturesDescribe(t *testing.T) {
	d := &fakeDriver{
		name:  "d",
		match: true,
		describe: func(info DeviceInfo) DeviceInfo {
			info.Model = "decorated"
			return info
		},
	}
	defer withScanner([]DeviceInfo{{DevicePath: "x"}}, nil)()
	m := NewManager(WithDrivers(d))
	got, _ := m.Scan()
	if got[0].Info.Model != "decorated" {
		t.Fatalf("Model = %v, want \"decorated\"", got[0].Info.Model)
	}
}

func TestManager_ScannedDeviceOpenUsesClaimedDriver(t *testing.T) {
	d := &fakeDriver{name: "track", match: true}
	defer withScanner([]DeviceInfo{{DevicePath: "x"}}, nil)()
	m := NewManager(WithDrivers(d))
	got, _ := m.Scan()
	dev, err := got[0].Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if dev == nil {
		t.Fatal("Open returned nil device")
	}
	_ = dev.Close()
	if d.opened != 1 {
		t.Fatalf("driver.Open called %d times, want 1", d.opened)
	}
	if d.gotInfo.DevicePath != "x" {
		t.Fatalf("driver.Open info = %+v, want DevicePath=x", d.gotInfo)
	}
}

func TestManager_ScannedDeviceOpenPropagatesOpenError(t *testing.T) {
	wantErr := errors.New("open failed")
	defer withScanner([]DeviceInfo{{DevicePath: "x"}}, nil)()
	m := NewManager(WithDrivers(&openErrorDriver{err: wantErr}))
	got, _ := m.Scan()
	_, err := got[0].Open()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Open: got %v, want %v", err, wantErr)
	}
}

func TestManager_ScanPropagatesScannerError(t *testing.T) {
	wantErr := errors.New("scanner failed")
	defer withScanner(nil, wantErr)()
	m := NewManager(WithDrivers(&fakeDriver{name: "any", match: true}))
	_, err := m.Scan()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Scan: got %v, want %v", err, wantErr)
	}
}

type stubDevice struct {
	info DeviceInfo
}

func (s stubDevice) Info() DeviceInfo       { return s.info }
func (s stubDevice) Vibrate(Command) error { return nil }
func (s stubDevice) Stop(Target) error     { return nil }
func (s stubDevice) StopAll() error        { return nil }
func (s stubDevice) Close() error          { return nil }

type openErrorDriver struct {
	err error
}

func (d *openErrorDriver) Match(DeviceInfo) bool           { return true }
func (d *openErrorDriver) Describe(i DeviceInfo) DeviceInfo { return i }
func (d *openErrorDriver) Open(DeviceInfo) (Device, error) { return nil, d.err }